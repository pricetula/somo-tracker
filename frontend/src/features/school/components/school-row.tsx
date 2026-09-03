"use client";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuSub,
    DropdownMenuSubTrigger,
    DropdownMenuPortal,
    DropdownMenuSubContent,
} from "@/components/ui/dropdown-menu";
import { Badge } from "@/components/ui/badge";
import { Pencil, Trash2 } from "lucide-react";
import { type SchoolWithMemberCount } from "../types";
import * as React from "react";

import { EditSchoolDialog } from "./edit-school-dialog";
import { DeleteSchoolAlert } from "./delete-school-alert";

export function SchoolRow({
    school,
    activeSchoolId,
    isSwitching,
    onSwitch,
}: {
    school: SchoolWithMemberCount;
    activeSchoolId: string;
    isSwitching: boolean;
    onSwitch: (id: string) => void;
}) {
    const [editOpen, setEditOpen] = React.useState(false);
    const [deleteOpen, setDeleteOpen] = React.useState(false);
    const isActive = school.id === activeSchoolId;

    return (
        <>
            <DropdownMenuSub>
                <DropdownMenuSubTrigger
                    disabled={isActive || isSwitching}
                    className="flex items-center gap-2"
                >
                    <Avatar className="h-7 w-7 overflow-hidden bg-transparent">
                        <AvatarFallback className="rounded-lg">
                            {school.name
                                .split(" ")
                                .map((n) => n.charAt(0))
                                .join("")
                                .toUpperCase()
                                .slice(0, 2)}
                        </AvatarFallback>
                    </Avatar>
                    <span className="flex-1 truncate">{school.name}</span>
                    {isActive && (
                        <Badge
                            variant="secondary"
                            className="bg-sky-100 text-[10px] text-sky-700 dark:bg-sky-900/30 dark:text-sky-400"
                        >
                            Active
                        </Badge>
                    )}
                </DropdownMenuSubTrigger>
                <DropdownMenuPortal>
                    <DropdownMenuSubContent>
                        <DropdownMenuItem
                            onClick={() => {
                                onSwitch(school.id);
                            }}
                        >
                            Switch to School
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => setEditOpen(true)}>
                            <Pencil className="mr-2 size-3.5" />
                            Edit Name
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                            onClick={() => setDeleteOpen(true)}
                            className="text-destructive focus:text-destructive"
                        >
                            <Trash2 className="mr-2 size-3.5" />
                            Delete School
                        </DropdownMenuItem>
                    </DropdownMenuSubContent>
                </DropdownMenuPortal>
            </DropdownMenuSub>

            <EditSchoolDialog school={school} open={editOpen} onOpenChange={setEditOpen} />
            <DeleteSchoolAlert school={school} open={deleteOpen} onOpenChange={setDeleteOpen} />
        </>
    );
}
