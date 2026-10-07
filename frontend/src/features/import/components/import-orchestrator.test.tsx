import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ImportOrchestrator } from "./import-orchestrator";

const createWrapper = () => {
    const queryClient = new QueryClient({
        defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const Wrapper = ({ children }: { children: React.ReactNode }) => (
        <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    Wrapper.displayName = "QueryClientWrapper";
    return Wrapper;
};

// Mock EventSource
class MockEventSource {
    url: string;
    private listeners: Record<string, ((e: MessageEvent) => void)[]> = {};
    public static instances: MockEventSource[] = [];
    constructor(url: string) {
        this.url = url;
        MockEventSource.instances.push(this);
    }
    addEventListener(type: string, cb: (e: MessageEvent) => void) {
        (this.listeners[type] ||= []).push(cb);
    }
    removeEventListener(_type: string, _cb: (e: MessageEvent) => void) {}
    close() {}
    emit(type: string, data: string) {
        const e = new MessageEvent(type, { data });
        (this.listeners[type] || []).forEach((cb) => cb(e));
    }
}

global.EventSource = MockEventSource as unknown as typeof EventSource;

describe("ImportOrchestrator", () => {
    beforeEach(() => {
        MockEventSource.instances = [];
    });

    it("renders initial import UI without crashing", async () => {
        const onMappedList = vi.fn(async () => ({
            job_id: "job-123",
            total_records: 2,
            status: "QUEUED",
        }));
        render(
            <ImportOrchestrator
                fieldDef={[]}
                onMappedList={onMappedList}
                progressUrl={(id) => `http://test/${id}`}
                showProgress
            />,
            { wrapper: createWrapper() }
        );

        // Initial UI
        expect(screen.getByRole("button", { name: /Upload/i })).toBeInTheDocument();
    });
});
