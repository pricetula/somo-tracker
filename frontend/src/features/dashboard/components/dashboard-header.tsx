import { Separator } from "@/components/ui/separator";

export function DashboardHeader() {
    const name = "Price Tula";
    const role = "Admin";
    const academicYear = "2026";
    const academicTermName = "Term 3";
    return (
        <header className="mb-8 flex justify-between">
            <div className="flex items-center gap-4">
                <h1 className="m-0 space-x-1 p-0 text-lg">
                    <span className="text-muted-foreground">Hi</span>
                    <span>{name}</span>
                </h1>
                <Separator orientation="vertical" />
                <p className="">{role}</p>
            </div>
            <div>
                <div className="text-base">{academicYear}</div>
                <div className="text-muted-foreground">{academicTermName}</div>
            </div>
        </header>
    );
}
