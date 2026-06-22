/**
 * Mock EventSource implementation for SSE testing.
 *
 * Replaces the global EventSource constructor with a mock that
 * records instances and provides emit/triggerError helpers.
 */

import { vi } from "vitest";

export class MockEventSource {
    static instances: MockEventSource[] = [];
    url: string;
    withCredentials = false;
    readyState = 0;
    onmessage: ((e: MessageEvent) => void) | null = null;
    onerror: ((e: Event) => void) | null = null;
    onopen: ((e: Event) => void) | null = null;
    close = vi.fn();

    constructor(url: string) {
        this.url = url;
        MockEventSource.instances.push(this);
    }

    emit(type: string, data: Record<string, unknown>) {
        const eventData = JSON.stringify({ type, ...data });
        this.onmessage?.({ data: eventData } as MessageEvent);
    }

    triggerError() {
        this.onerror?.(new Event("error"));
    }

    static reset() {
        MockEventSource.instances = [];
    }
}

// Replace global EventSource
global.EventSource = MockEventSource as unknown as typeof EventSource;
