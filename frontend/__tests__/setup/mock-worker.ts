/**
 * Mock Worker factory for testing Web Worker interactions.
 *
 * Creates a Worker-like mock that captures postMessage calls and
 * lets tests trigger onmessage manually.
 */

import { vi } from "vitest";

export interface MockWorkerInstance {
    postMessage: ReturnType<typeof vi.fn>;
    onmessage: ((e: MessageEvent) => void) | null;
    terminate: ReturnType<typeof vi.fn>;
    triggerOnMessage(data: unknown): void;
}

/**
 * Creates a mock Worker constructor that returns controlled instances.
 */
export function createMockWorkerConstructor(): {
    Worker: new (url: string | URL) => MockWorkerInstance;
    instances: MockWorkerInstance[];
} {
    const instances: MockWorkerInstance[] = [];

    const MockWorker = class {
        postMessage = vi.fn();
        onmessage: ((e: MessageEvent) => void) | null = null;
        terminate = vi.fn();
        url: string;

        constructor(url: string | URL) {
            this.url = typeof url === "string" ? url : url.href;
            instances.push(this);
        }

        triggerOnMessage(data: unknown) {
            this.onmessage?.({ data } as MessageEvent);
        }
    } as unknown as new (url: string | URL) => MockWorkerInstance;

    return { Worker: MockWorker, instances };
}
