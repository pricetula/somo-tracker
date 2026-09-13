"use client";

import { useEffect, useState } from "react";

export default function TestSsePage() {
    const [log, setLog] = useState<string[]>([]);
    const [jobId, setJobId] = useState("");

    useEffect(() => {
        if (!jobId) return;
        const url = `/backend/api/admins/invitations/jobs/${jobId}/events`;
        const es = new EventSource(url);
        es.onopen = () => setLog((l) => [...l, `Connecting to ${url}`]);
        es.onopen = () => setLog((l) => [...l, "EventSource open"]);
        es.onmessage = (e) => setLog((l) => [...l, `message: ${e.data}`]);
        es.addEventListener("progress", (e) => setLog((l) => [...l, `progress: ${e.data}`]));
        es.addEventListener("heartbeat", () => setLog((l) => [...l, "heartbeat"]));
        es.onerror = (e) => setLog((l) => [...l, `error: ${e}`]);
        return () => es.close();
    }, [jobId]);

    return (
        <div className="space-y-4 p-6">
            <h1 className="text-xl font-bold">SSE Test</h1>
            <input
                placeholder="Job ID"
                value={jobId}
                onChange={(e) => setJobId(e.target.value)}
                className="rounded border p-2"
            />
            <pre className="bg-muted h-96 overflow-auto rounded p-4 text-sm">{log.join("\n")}</pre>
        </div>
    );
}
