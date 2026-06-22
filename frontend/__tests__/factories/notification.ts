/**
 * Factory for building import notification objects in tests.
 */

export interface ImportNotification {
    id: string;
    importJobId: string;
    tenantId: string;
    successCount: number;
    failedCount: number;
    createdAt: number;
}

let notifCounter = 0;

export function buildNotification(overrides?: Partial<ImportNotification>): ImportNotification {
    notifCounter++;
    return {
        id: `notif-${notifCounter}`,
        importJobId: `job-${String(notifCounter).padStart(3, "0")}`,
        tenantId: "tenant-abc",
        successCount: 95,
        failedCount: 0,
        createdAt: Date.now(),
        ...overrides,
    };
}

export function resetNotifCounter() {
    notifCounter = 0;
}
