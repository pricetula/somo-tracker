/**
 * Shared column wizard step wrapper.
 *
 * Provides a consistent layout: explainer, preview slot, column selector, confirm/back actions.
 */

"use client";

import * as React from "react";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";

// ─── Types ─────────────────────────────────────────────────────────────────

export interface WizardStepProps {
    explainer: string;
    previewValue: string;
    placeholder: string;
    /** One or more column select slots */
    selectSlots: SelectSlotConfig[];
    claimedColumns: Set<string>;
    onConfirm: () => void;
    onBack?: () => void;
    /** If true, shows a Skip action instead of forcing confirm */
    showSkip?: boolean;
    onSkip?: () => void;
    /** Confirm button label (default "Confirm") */
    confirmLabel?: string;
    /** If confirm is disabled */
    confirmDisabled?: boolean;
}

export interface SelectSlotConfig {
    id: string;
    label: string;
    value: string;
    options: { value: string; label: string }[];
    onChange: (value: string) => void;
    onRemove?: () => void;
    canRemove?: boolean;
    onAdd?: () => void;
    showAdd?: boolean;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ColumnWizardStep({
    explainer,
    previewValue,
    placeholder,
    selectSlots,
    claimedColumns,
    onConfirm,
    onBack,
    showSkip,
    onSkip,
    confirmLabel = "Confirm",
    confirmDisabled = false,
}: WizardStepProps) {
    return (
        <div className="space-y-4">
            {/* Explainer */}
            <p className="text-muted-foreground text-sm">{explainer}</p>

            {/* Preview */}
            <div className="bg-muted/20 rounded-md px-3 py-2">
                <p className="text-muted-foreground text-xs">Preview:</p>
                <p className="mt-0.5 text-sm font-medium">
                    {previewValue || (
                        <span className="text-muted-foreground/50">{placeholder}</span>
                    )}
                </p>
            </div>

            {/* Select slots */}
            <div className="space-y-3">
                {selectSlots.map((slot) => (
                    <div key={slot.id} className="flex items-center gap-2">
                        <div className="flex-1">
                            <label className="text-muted-foreground mb-1 block text-xs">
                                {slot.label}
                            </label>
                            <Select value={slot.value} onValueChange={slot.onChange}>
                                <SelectTrigger className="h-9 text-sm">
                                    <SelectValue placeholder="Select column…" />
                                </SelectTrigger>
                                <SelectContent>
                                    {slot.options.map((opt) => {
                                        const isClaimed =
                                            claimedColumns.has(opt.value) &&
                                            opt.value !== slot.value;
                                        return (
                                            <SelectItem
                                                key={opt.value}
                                                value={opt.value}
                                                disabled={isClaimed}
                                                className={isClaimed ? "opacity-50" : ""}
                                            >
                                                {opt.label}
                                                {isClaimed && (
                                                    <span className="text-muted-foreground ml-2 text-xs">
                                                        (mapped elsewhere)
                                                    </span>
                                                )}
                                            </SelectItem>
                                        );
                                    })}
                                </SelectContent>
                            </Select>
                        </div>
                        {slot.canRemove && slot.onRemove && (
                            <button
                                onClick={slot.onRemove}
                                className="text-muted-foreground hover:text-foreground mt-5 text-xs"
                            >
                                Remove
                            </button>
                        )}
                    </div>
                ))}

                {/* Add column button */}
                {selectSlots.length > 0 &&
                    selectSlots[selectSlots.length - 1].showAdd &&
                    selectSlots[selectSlots.length - 1].onAdd && (
                        <button
                            onClick={selectSlots[selectSlots.length - 1].onAdd}
                            className="text-muted-foreground hover:text-foreground text-xs font-medium"
                        >
                            (+) Add Column Component
                        </button>
                    )}
            </div>

            {/* Actions */}
            <div className="flex items-center justify-between pt-2">
                <div>
                    {onBack && (
                        <button
                            onClick={onBack}
                            className="text-muted-foreground hover:text-foreground text-sm"
                        >
                            Back
                        </button>
                    )}
                </div>
                <div className="flex items-center gap-2">
                    {showSkip && onSkip && (
                        <button
                            onClick={onSkip}
                            className="text-muted-foreground hover:text-foreground rounded-md px-3 py-1.5 text-sm"
                        >
                            Skip Column Mapping
                        </button>
                    )}
                    <button
                        onClick={onConfirm}
                        disabled={confirmDisabled}
                        className="bg-primary text-primary-foreground hover:bg-primary/90 rounded-md px-4 py-1.5 text-sm font-medium disabled:opacity-50"
                    >
                        {confirmLabel}
                    </button>
                </div>
            </div>
        </div>
    );
}
