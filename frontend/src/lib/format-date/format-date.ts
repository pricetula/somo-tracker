import { format } from "date-fns";

export function formatDate(dateStr: string) {
    if (!dateStr) return "";
    let localFormattedDate = getLocalStorageFormattedDate();
    if (!localFormattedDate) {
        localFormattedDate = "MMM d, yyyy";
    }
    return format(dateStr, localFormattedDate);
}

export function getLocalStorageFormattedDate() {
    return localStorage.getItem("SET_DATE_FORMAT_TYPE");
}

export function setLocalStorageFormattedDate(format: string) {
    return localStorage.setItem("SET_DATE_FORMAT_TYPE", format);
}
