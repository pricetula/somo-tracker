export interface FinanceListItem {
    membership_id: string;
    user_id: string;
    email: string;
    full_name: string;
    invited_at?: string | null;
    accepted_at?: string | null;
    is_active: boolean;
    created_at: string;
}
