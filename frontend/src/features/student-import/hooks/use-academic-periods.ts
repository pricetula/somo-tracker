/**
 * Hook for fetching academic years and periods from the backend.
 *
 * Academic years are loaded on mount. Once a year is selected, periods (terms)
 * for that year are fetched.
 */

"use client";

import * as React from "react";
import { getErrorMessage } from "@/lib/errors";
import { fetchAcademicYears, fetchAcademicPeriods } from "../services/import-api";
import type { AcademicYearRecord, AcademicPeriodRecord } from "../types";

export interface AcademicYearsState {
    years: AcademicYearRecord[];
    yearsLoading: boolean;
    yearsError: string | null;
    retryYears: () => void;
}

export function useAcademicYears(): AcademicYearsState {
    const [years, setYears] = React.useState<AcademicYearRecord[]>([]);
    const [yearsError, setYearsError] = React.useState<string | null>(null);
    const [yearsLoading, setYearsLoading] = React.useState(true);
    const [retryCount, setRetryCount] = React.useState(0);

    React.useEffect(() => {
        let cancelled = false;

        fetchAcademicYears()
            .then((data) => {
                if (cancelled) return;
                setYears(data);
                setYearsError(null);
                setYearsLoading(false);
            })
            .catch((err: unknown) => {
                if (cancelled) return;
                setYears([]);
                setYearsError(getErrorMessage(err));
                setYearsLoading(false);
            });

        return () => {
            cancelled = true;
        };
    }, [retryCount]);

    const retryYears = React.useCallback(() => {
        setYearsLoading(true);
        setYearsError(null);
        setRetryCount((c) => c + 1);
    }, []);

    return { years, yearsLoading, yearsError, retryYears };
}

export interface AcademicPeriodsState {
    periods: AcademicPeriodRecord[];
    periodsLoading: boolean;
    periodsError: string | null;
    retryPeriods: () => void;
}

type PeriodsAction =
    | { type: "FETCH_START" }
    | { type: "FETCH_SUCCESS"; data: AcademicPeriodRecord[] }
    | { type: "FETCH_ERROR"; error: string };

interface PeriodsState {
    periods: AcademicPeriodRecord[];
    periodsLoading: boolean;
    periodsError: string | null;
}

function periodsReducer(state: PeriodsState, action: PeriodsAction): PeriodsState {
    switch (action.type) {
        case "FETCH_START":
            return { ...state, periodsLoading: true, periodsError: null };
        case "FETCH_SUCCESS":
            return { periods: action.data, periodsLoading: false, periodsError: null };
        case "FETCH_ERROR":
            return { periods: [], periodsLoading: false, periodsError: action.error };
    }
}

export function useAcademicPeriods(academicYearId: string | null): AcademicPeriodsState {
    const [retryCount, setRetryCount] = React.useState(0);

    const [state, dispatch] = React.useReducer(periodsReducer, {
        periods: [],
        periodsLoading: false,
        periodsError: null,
    });

    React.useEffect(() => {
        if (!academicYearId) return;

        dispatch({ type: "FETCH_START" });

        let cancelled = false;

        fetchAcademicPeriods(academicYearId)
            .then((data) => {
                if (cancelled) return;
                dispatch({ type: "FETCH_SUCCESS", data });
            })
            .catch((err: unknown) => {
                if (cancelled) return;
                dispatch({ type: "FETCH_ERROR", error: getErrorMessage(err) });
            });

        return () => {
            cancelled = true;
        };
    }, [academicYearId, retryCount]);

    const retryPeriods = React.useCallback(() => {
        setRetryCount((c) => c + 1);
    }, []);

    // Derive display values — when no year is selected, show empty / not loading
    const displayPeriods = academicYearId ? state.periods : [];
    const displayLoading = academicYearId ? state.periodsLoading : false;
    const displayError = academicYearId ? state.periodsError : null;

    return {
        periods: displayPeriods,
        periodsLoading: displayLoading,
        periodsError: displayError,
        retryPeriods,
    };
}
