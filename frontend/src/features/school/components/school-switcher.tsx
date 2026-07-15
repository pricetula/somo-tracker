/**
 * School Switcher — dropdown menu to view, switch, edit, and delete schools.
 *
 * Renders inside the sidebar header. Shows the current active school
 * and a dropdown list of all available schools in the tenant.
 */

"use client";

import * as React from "react";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuGroup,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
    DropdownMenuSub,
    DropdownMenuSubTrigger,
    DropdownMenuPortal,
    DropdownMenuSubContent,
} from "@/components/ui/dropdown-menu";
import {
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    useSidebar,
} from "@/components/ui/sidebar";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { ChevronsUpDownIcon, Plus, Pencil, Trash2 } from "lucide-react";

import { useMe } from "@/hooks/use-auth";
import {
    useSchools,
    useSetActiveSchool,
    useUpdateSchool,
    useDeleteSchool,
} from "../hooks/use-schools";
import { getErrorMessage } from "@/lib/errors";
import { CreateSchoolDialog } from "./create-school-dialog";
import type { SchoolWithMemberCount } from "../types";

// ─── Edit School Dialog ───────────────────────────────────────────────────

function EditSchoolDialog({
    school,
    open,
    onOpenChange,
}: {
    school: SchoolWithMemberCount;
    open: boolean;
    onOpenChange: (open: boolean) => void;
}) {
    const [name, setName] = React.useState(school.name);
    const updateMutation = useUpdateSchool();

    // Reset name when dialog opens — this is an initialization effect that syncs
    // external state (school data) into local state. The alternative would be
    // key-based remounting, which would reset scroll and focus state.
    React.useEffect(() => {
        // eslint-disable-next-line react-hooks/set-state-in-effect
        if (open) setName(school.name);
    }, [open, school.name]);

    async function handleSave() {
        if (!name.trim()) return;
        updateMutation.mutate(
            { id: school.id, payload: { name: name.trim() } },
            {
                onSuccess: () => onOpenChange(false),
            }
        );
    }

    return (
        <AlertDialog open={open} onOpenChange={onOpenChange}>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>Edit School</AlertDialogTitle>
                    <AlertDialogDescription>
                        Update the name for &ldquo;{school.name}&rdquo;
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <div className="space-y-2 py-2">
                    <Label htmlFor="edit-school-name">School Name</Label>
                    <Input
                        id="edit-school-name"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        autoFocus
                    />
                    {updateMutation.error && (
                        <p className="text-destructive text-sm">
                            {getErrorMessage(updateMutation.error)}
                        </p>
                    )}
                </div>
                <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction
                        onClick={handleSave}
                        disabled={!name.trim() || updateMutation.isPending}
                    >
                        {updateMutation.isPending ? "Saving…" : "Save"}
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    );
}

// ─── Delete School Alert ──────────────────────────────────────────────────

function DeleteSchoolAlert({
    school,
    open,
    onOpenChange,
}: {
    school: SchoolWithMemberCount;
    open: boolean;
    onOpenChange: (open: boolean) => void;
}) {
    const deleteMutation = useDeleteSchool();

    return (
        <AlertDialog open={open} onOpenChange={onOpenChange}>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>Delete School</AlertDialogTitle>
                    <AlertDialogDescription>
                        Are you sure you want to delete <strong>{school.name}</strong>? This cannot
                        be undone. All data associated with this school will be permanently removed.
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction
                        onClick={() => deleteMutation.mutate(school.id)}
                        disabled={deleteMutation.isPending}
                        className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    >
                        {deleteMutation.isPending ? "Deleting…" : "Delete"}
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    );
}

// ─── School Row (with edit/delete submenu) ────────────────────────────────

function SchoolRow({
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
                        <AvatarFallback className="rounded-lg text-xs">
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

// ─── Main Component ───────────────────────────────────────────────────────

export function SchoolSwitcher() {
    const { isMobile } = useSidebar();
    const { data: me } = useMe();
    const { data: schoolsData } = useSchools();
    const { mutate: switchSchool, isPending: isSwitching } = useSetActiveSchool();
    const [dialogOpen, setDialogOpen] = React.useState(false);

    const activeSchoolName = me?.school_name ?? "School";
    const activeSchoolId = me?.school_id ?? "";
    const schools = schoolsData?.items ?? [];

    const initials = activeSchoolName
        .split(" ")
        .map((n) => n.charAt(0))
        .join("")
        .toUpperCase()
        .slice(0, 2);

    return (
        <>
            <SidebarMenu>
                <SidebarMenuItem>
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <SidebarMenuButton
                                size="lg"
                                className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                            >
                                <Avatar className="h-8 w-8 overflow-hidden bg-transparent">
                                    <AvatarFallback className="rounded-lg">
                                        {initials}
                                    </AvatarFallback>
                                </Avatar>
                                <div className="grid flex-1 text-left leading-tight">
                                    <span className="truncate font-medium">{activeSchoolName}</span>
                                </div>
                                <ChevronsUpDownIcon className="ml-auto size-4" />
                            </SidebarMenuButton>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent
                            className="w-fit min-w-56"
                            side={isMobile ? "bottom" : "right"}
                            align="start"
                            sideOffset={4}
                        >
                            <DropdownMenuLabel className="text-muted-foreground text-xs font-medium">
                                Schools
                            </DropdownMenuLabel>
                            <DropdownMenuGroup>
                                {schools.length === 0 ? (
                                    <div className="text-muted-foreground px-2 py-3 text-center text-xs">
                                        No schools yet
                                    </div>
                                ) : (
                                    schools.map((school) => (
                                        <SchoolRow
                                            key={school.id}
                                            school={school}
                                            activeSchoolId={activeSchoolId}
                                            isSwitching={isSwitching}
                                            onSwitch={switchSchool}
                                        />
                                    ))
                                )}
                            </DropdownMenuGroup>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                                onClick={() => setDialogOpen(true)}
                                className="text-muted-foreground flex items-center gap-2"
                            >
                                <Plus className="size-4" />
                                <span>Add School</span>
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </SidebarMenuItem>
            </SidebarMenu>

            <CreateSchoolDialog open={dialogOpen} onOpenChange={setDialogOpen} />
        </>
    );
}
