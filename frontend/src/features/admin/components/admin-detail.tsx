interface AdminDetailProps {
    id: string;
}

export function AdminDetail({ id }: AdminDetailProps) {
    return <article>{id}</article>;
}
