'use client'

import { useState } from 'react'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import { cn } from 'cn'
import type { BulkAction } from './row-actions'

interface DataTableBulkBarProps {
  selectedCount: number
  selectedIds: string[]
  bulkActions: BulkAction[]
  onActionSelect: (actionId: string, ids: string[]) => void
  onClearSelection: () => void
}

export function DataTableBulkBar({
  selectedCount,
  selectedIds,
  bulkActions,
  onActionSelect,
  onClearSelection,
}: DataTableBulkBarProps) {
  const [openActionId, setOpenActionId] = useState<string | null>(null)

  const handleActionClick = (action: BulkAction) => {
    setOpenActionId(action.id)
  }

  const handleConfirm = (action: BulkAction) => {
    onActionSelect(action.id, selectedIds)
    setOpenActionId(null)
  }

  return (
    <>
      <div
        className={cn(
          'flex items-center justify-between px-4 py-3 bg-muted border-b border-border',
          selectedCount === 0 && 'hidden'
        )}
      >
        <span className="text-sm font-medium text-foreground">
          {selectedCount} selected
        </span>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={onClearSelection}
            className="text-xs"
          >
            Clear selection
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger>
              <Button variant="outline" size="sm" className="text-xs">
                Bulk actions
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuLabel>Actions</DropdownMenuLabel>
              {bulkActions.map((action: BulkAction) => (
                <DropdownMenuItem
                  key={action.id}
                  className={cn(
                    action.variant === 'destructive' && 'text-destructive'
                  )}
                  onClick={() => handleActionClick(action)}
                >
                  {action.label}
                </DropdownMenuItem>
              ))}
              <DropdownMenuSeparator />
              <DropdownMenuItem
                className="text-xs text-muted-foreground"
                onClick={onClearSelection}
              >
                Clear selection
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {bulkActions.length > 0 && selectedCount > 0 && (
        <>
          {bulkActions.map((action: BulkAction) => (
            <AlertDialog
              key={action.id}
              open={openActionId === action.id}
              onOpenChange={(open) => {
                if (!open) setOpenActionId(null)
              }}
            >
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>{action.confirm.title}</AlertDialogTitle>
                  <AlertDialogDescription>
                    {action.confirm.description(selectedCount)}
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={() => handleConfirm(action)}
                    variant={action.variant === 'destructive' ? 'destructive' : undefined}
                  >
                    {action.confirm.confirmLabel}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          ))}
        </>
      )}
    </>
  )
}
