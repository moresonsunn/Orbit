type MessageCallback = (payload: any) => void;

class WebSocketClient {
  private ws: WebSocket | null = null;
  private subscribers: Map<string, Set<MessageCallback>> = new Map();
  private activeTopics: Set<string> = new Set();
  private reconnectTimeout = 1000;
  private maxReconnectTimeout = 15000;
  private shouldReconnect = true;
  private connected = false;
  private statusListeners: Set<(connected: boolean) => void> = new Set();

  constructor() {
    this.connect();
  }

  public onStatusChange(callback: (connected: boolean) => void) {
    this.statusListeners.add(callback);
    callback(this.connected);
    return () => {
      this.statusListeners.delete(callback);
    };
  }

  public subscribe(topic: string, callback: MessageCallback) {
    if (!this.subscribers.has(topic)) {
      this.subscribers.set(topic, new Set());
    }
    this.subscribers.get(topic)!.add(callback);

    if (!this.activeTopics.has(topic)) {
      this.activeTopics.add(topic);
      this.sendAction('subscribe', topic);
    }

    return () => {
      const subs = this.subscribers.get(topic);
      if (subs) {
        subs.delete(callback);
        if (subs.size === 0) {
          this.subscribers.delete(topic);
          this.activeTopics.delete(topic);
          this.sendAction('unsubscribe', topic);
        }
      }
    };
  }

  private connect() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws`;

    try {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        this.connected = true;
        this.reconnectTimeout = 1000;
        this.notifyStatus(true);

        // Resubscribe active topics
        this.activeTopics.forEach((topic) => {
          this.sendAction('subscribe', topic);
        });
      };

      this.ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.topic && this.subscribers.has(msg.topic)) {
            const subs = this.subscribers.get(msg.topic)!;
            subs.forEach((cb) => cb(msg.payload));
          }
        } catch {
          // ignore invalid json
        }
      };

      this.ws.onclose = () => {
        this.connected = false;
        this.notifyStatus(false);
        if (this.shouldReconnect) {
          setTimeout(() => this.connect(), this.reconnectTimeout);
          this.reconnectTimeout = Math.min(this.reconnectTimeout * 1.5, this.maxReconnectTimeout);
        }
      };

      this.ws.onerror = () => {
        this.ws?.close();
      };
    } catch {
      this.connected = false;
      this.notifyStatus(false);
      setTimeout(() => this.connect(), this.reconnectTimeout);
    }
  }

  private sendAction(action: 'subscribe' | 'unsubscribe', topic: string) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ action, topic }));
    }
  }

  private notifyStatus(connected: boolean) {
    this.statusListeners.forEach((cb) => cb(connected));
  }

  public close() {
    this.shouldReconnect = false;
    this.ws?.close();
  }
}

export const wsClient = new WebSocketClient();
