/**
 * Link Parent Dialog — modal to search and link a parent to a student.
 *
 * Used from the student detail page to assign a guardian/parent.
 * Features:
 * - Search parents by name or email
 * - Select relationship type (optional)
 * - Toggle primary guardian status
 */

"use client";

import * as React from "react";

import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Loader2, Search } from "lucide-react";

import { useLinkStudent } from "../hooks/use-parents";
import { useParents } from "../hooks/use-parents";
import { getErrorMessage } from "@/lib/errors";

// ─── Props ─────────────────────────────────────────────────────────────────

interface LinkParentDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    studentId: string;
}

const RELATIONSHIP_OPTIONS = [
    "Mother",
    "Father",
    "Guardian",
    "Grandparent",
    "Sibling",
    "Aunt",
    "Uncle",
    "Other",
] as const;

// ─── Component ─────────────────────────────────────────────────────────────

export function LinkParentDialog({ open, onOpenChange, studentId }: LinkParentDialogProps) {
    const [search, setSearch] = React.useState("");
    const [selectedParentId, setSelectedParentId] = React.useState("");
    const [relationship, setRelationship] = React.useState("");
    const [isPrimary, setIsPrimary] = React.useState(false);
    const [error, setError] = React.useState<string | null>(null);

    const linkStudent = useLinkStudent();

    const { data: parentsData, isLoading: parentsLoading } = useParents(
        { search: search || undefined, limit: 50 },
        { enabled: open }
    );

    const parents = parentsData?.items ?? [];

    const handleLink = async () => {
        if (!selectedParentId) {
            setError("Please select a parent");
            return;
        }

        setError(null);

        try {
            await linkStudent.mutateAsync({
                parentId: selectedParentId,
                data: {
                    student_id: studentId,
                    relationship: relationship || null,
                    is_primary: isPrimary || undefined,
                },
            });
            onOpenChange(false);
        } catch (err) {
            setError(getErrorMessage(err));
        }
    };

    const handleOpenChange = (next: boolean) => {
        if (next) {
            setSearch("");
            setSelectedParentId("");
            setRelationship("");
            setIsPrimary(false);
            setError(null);
        }
        onOpenChange(next);
    };

    const selectedParent = parents.find((p) => p.id === selectedParentId);

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="max-w-md">
                <DialogHeader>
                    <DialogTitle>Link Parent</DialogTitle>
                    <DialogDescription>
                        Search and select a parent to link to this student.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4">
                    {error && (
                        <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2">
                            {error}
                        </div>
                    )}

                    {/* Parent search */}
                    <div className="space-y-1.5">
                        <Label>Search Parent</Label>
                        <div className="relative">
                            <Search className="text-muted-foreground absolute top-2.5 left-2.5 size-4" />
                            <Input
                                placeholder="Type parent name or email…"
                                value={search}
                                onChange={(e) => {
                                    setSearch(e.target.value);
                                    setSelectedParentId("");
                                }}
                                className="pl-8"
                            />
                        </div>
                    </div>

                    {/* Parent results */}
                    <div className="min-h-30">
                        {parentsLoading ? (
                            <div className="flex items-center justify-center py-8">
                                <Loader2 className="text-muted-foreground size-5 animate-spin" />
                            </div>
                        ) : parents.length === 0 ? (
                            <p className="text-muted-foreground py-4 text-center">
                                {search
                                    ? "No parents match your search"
                                    : "Type to search for parents"}
                            </p>
                        ) : (
                            <div className="border-border/40 max-h-48 space-y-1 overflow-auto rounded-md border">
                                {parents.map((p) => {
                                    const isSelected = p.id === selectedParentId;
                                    return (
                                        <button
                                            key={p.id}
                                            type="button"
                                            className={`hover:bg-muted/50 flex w-full items-center gap-2 px-3 py-2 text-left transition-colors ${
                                                isSelected ? "bg-muted font-medium" : ""
                                            }`}
                                            onClick={() => setSelectedParentId(p.id)}
                                        >
                                            <span className="flex-1 truncate">{p.full_name}</span>
                                            <span className="text-muted-foreground truncate">
                                                {p.email}
                                            </span>
                                            {isSelected && (
                                                <Badge
                                                    variant="secondary"
                                                    className="bg-sky-100 text-[10px] text-sky-700 dark:bg-sky-900/30 dark:text-sky-400"
                                                >
                                                    Selected
                                                </Badge>
                                            )}
                                        </button>
                                    );
                                })}
                            </div>
                        )}
                    </div>

                    {/* Selected parent summary */}
                    {selectedParent && (
                        <div className="bg-muted/30 rounded-md px-3 py-2">
                            <span className="font-medium">Selected: </span>
                            {selectedParent.full_name}
                            <span className="text-muted-foreground"> — {selectedParent.email}</span>
                        </div>
                    )}

                    {/* Relationship */}
                    <div className="space-y-1.5">
                        <Label htmlFor="relationship">Relationship (optional)</Label>
                        <Select value={relationship} onValueChange={setRelationship}>
                            <SelectTrigger id="relationship">
                                <SelectValue placeholder="Select relationship" />
                            </SelectTrigger>
                            <SelectContent>
                                {RELATIONSHIP_OPTIONS.map((r) => (
                                    <SelectItem key={r} value={r}>
                                        {r}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </div>

                    {/* Is Primary */}
                    <div className="flex items-center gap-3">
                        <Switch
                            id="is_primary"
                            checked={isPrimary}
                            onCheckedChange={setIsPrimary}
                        />
                        <Label htmlFor="is_primary" className="">
                            Primary guardian
                        </Label>
                    </div>

                    {/* Actions */}
                    <div className="flex items-center justify-end gap-3 pt-2">
                        <Button
                            variant="ghost"
                            onClick={() => onOpenChange(false)}
                            disabled={linkStudent.isPending}
                        >
                            Cancel
                        </Button>
                        <Button
                            onClick={handleLink}
                            disabled={!selectedParentId || linkStudent.isPending}
                        >
                            {linkStudent.isPending ? (
                                <>
                                    <Loader2 className="mr-1.5 size-4 animate-spin" />
                                    Linking…
                                </>
                            ) : (
                                "Link Parent"
                            )}
                        </Button>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
}
