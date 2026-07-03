/**
 * Mock EventSource implementation for SSE testing.
 *
 * Replaces the global EventSource constructor with a mock that
 * records instances and provides emit/triggerError helpers.
 */

import { vi } from "vitest";

type EventListenerMap = Map<string, Set<(e: MessageEvent) => void>>;

export class MockEventSource {
    static instances: MockEventSource[] = [];
    url: string;
    withCredentials = false;
    readyState = 0;
    onmessage: ((e: MessageEvent) => void) | null = null;
    onerror: ((e: Event) => void) | null = null;
    onopen: ((e: Event) => void) | null = null;
    close = vi.fn();

    private _listeners: EventListenerMap = new Map();
    private _eventCount = 0;

    CONNECTING = 0;
    OPEN = 1;
    CLOSED = 2;

    constructor(url: string) {
        this.url = url;
        this.readyState = this.CONNECTING;
        MockEventSource.instances.push(this);
        // Simulate open after microtask
        setTimeout(() => {
            this.readyState = this.OPEN;
            this.onopen?.(new Event("open"));
        }, 0);
    }

    addEventListener(type: string, listener: (e: MessageEvent) => void) {
        if (!this._listeners.has(type)) {
            this._listeners.set(type, new Set());
        }
        this._listeners.get(type)!.add(listener);
    }

    removeEventListener(type: string, listener: (e: MessageEvent) => void) {
        this._listeners.get(type)?.delete(listener);
    }

    dispatchEvent(event: Event): boolean {
        const type = event.type;
        const listeners = this._listeners.get(type);
        if (listeners) {
            for (const listener of listeners) {
                listener(event as MessageEvent);
            }
        }
        // Also call onmessage/onerror/onopen
        if (type === "message" && this.onmessage) {
            this.onmessage(event as MessageEvent);
        } else if (type === "error" && this.onerror) {
            this.onerror(event);
        } else if (type === "open" && this.onopen) {
            this.onopen(event);
        }
        return true;
    }

    /**
     * Emit a structured event (type + data) to all registered listeners.
     */
    emit(type: string, data: Record<string, unknown>) {
        const eventData = JSON.stringify(data);
        const event = new MessageEvent(type, { data: eventData });
        this.dispatchEvent(event);
    }

    /**
     * Trigger an error event.
     */
    triggerError() {
        this.onerror?.(new Event("error"));
        // Also dispatch to addEventListener listeners
        const listeners = this._listeners.get("error");
        if (listeners) {
            for (const listener of listeners) {
                listener(new Event("error") as unknown as MessageEvent);
            }
        }
    }

    static reset() {
        MockEventSource.instances = [];
    }
}

// Replace global EventSource
global.EventSource = MockEventSource as unknown as typeof EventSource;
