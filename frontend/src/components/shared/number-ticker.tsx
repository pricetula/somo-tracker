"use client";

import { useEffect, useRef } from "react";
import { useInView, useMotionValue, useSpring } from "framer-motion";

interface NumberTickerProps {
    value: number;
    direction?: "up" | "down";
    delay?: number; // delay in seconds
    className?: string;
    decimalPlaces?: number;
}

export function NumberTicker({
    value,
    direction = "up",
    delay = 0,
    className = "",
    decimalPlaces = 0,
}: NumberTickerProps) {
    const ref = useRef<HTMLSpanElement>(null);
    const motionValue = useMotionValue(direction === "down" ? value : 0);
    const springValue = useSpring(motionValue, {
        damping: 30,
        stiffness: 100,
    });
    const isInView = useInView(ref, { once: true, margin: "0px" });

    useEffect(() => {
        if (isInView) {
            const timer = setTimeout(() => {
                motionValue.set(value);
            }, delay * 1000);
            return () => clearTimeout(timer);
        }
    }, [motionValue, isInView, delay, value]);

    useEffect(() => {
        return springValue.on("change", (latest) => {
            if (ref.current) {
                ref.current.textContent = Intl.NumberFormat("en-US", {
                    minimumFractionDigits: decimalPlaces,
                    maximumFractionDigits: decimalPlaces,
                }).format(latest);
            }
        });
    }, [springValue, decimalPlaces]);

    return <span className={`inline-block tracking-wider tabular-nums ${className}`} ref={ref} />;
}
