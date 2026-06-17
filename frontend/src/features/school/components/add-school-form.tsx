"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { EducationSystemCombobox } from "@/features/education-system";
import { useCreateSchool } from "@/hooks/use-schools";

interface AddSchoolFormProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

export function AddSchoolForm({ open, onOpenChange }: AddSchoolFormProps) {
    const router = useRouter();
    const createSchool = useCreateSchool();
    const [name, setName] = React.useState("");
    const [educationSystemId, setEducationSystemId] = React.useState("");

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!name.trim()) {
            toast.error("School name is required");
            return;
        }

        if (!educationSystemId) {
            toast.error("Please select an education system");
            return;
        }

        createSchool.mutate(
            {
                name: name.trim(),
                education_system_id: educationSystemId,
            },
            {
                onSuccess: () => {
                    setName("");
                    setEducationSystemId("");
                    onOpenChange(false);
                    router.refresh();
                },
            }
        );
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Add School</DialogTitle>
                    <DialogDescription>
                        Create a new school under your organization.
                    </DialogDescription>
                </DialogHeader>
                <form onSubmit={handleSubmit} className="space-y-4">
                    <div className="space-y-2">
                        <Label htmlFor="school-name">School Name</Label>
                        <Input
                            id="school-name"
                            placeholder="Enter school name"
                            value={name}
                            onChange={(e) => setName(e.target.value)}
                            disabled={createSchool.isPending}
                            autoFocus
                        />
                    </div>
                    <div className="space-y-2">
                        <Label htmlFor="education-system">Education System</Label>
                        <EducationSystemCombobox
                            value={educationSystemId}
                            onValueChange={setEducationSystemId}
                            disabled={createSchool.isPending}
                        />
                    </div>
                    <div className="flex justify-end gap-2">
                        <Button
                            type="button"
                            variant="outline"
                            onClick={() => onOpenChange(false)}
                            disabled={createSchool.isPending}
                        >
                            Cancel
                        </Button>
                        <Button type="submit" disabled={createSchool.isPending}>
                            {createSchool.isPending ? "Creating..." : "Create School"}
                        </Button>
                    </div>
                </form>
            </DialogContent>
        </Dialog>
    );
}
