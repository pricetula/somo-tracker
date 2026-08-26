import { formatDate, parse, isMatch, isValid } from "date-fns";

interface OptionFormatDateString {
    inputFormat?: string;
    outputFormat: string;
}

const currentDateObj = new Date();

export function formatDateString(dateStr: string, options: OptionFormatDateString): string {
    try {
        if (!options.inputFormat || !options.outputFormat || !isMatch(dateStr, options.inputFormat))
            return "";

        const dateObj = parse(dateStr, options.inputFormat, currentDateObj);

        return formatDateObject(dateObj, options.outputFormat);
    } catch {
        return "";
    }
}

export function formatDateObject(dateObj: Date, outputFormat: string): string {
    try {
        if (!isValid(dateObj)) return "";

        return formatDate(dateObj, outputFormat);
    } catch {
        return "";
    }
}
