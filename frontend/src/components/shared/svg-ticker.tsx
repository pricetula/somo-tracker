"use client";

import { useEffect } from "react";
import { motion, useMotionValue, useSpring, useTransform } from "framer-motion";

interface SvgNumberTickerProps {
    value: number;
    x?: number;
    y?: number;
    className?: string;
    decimalPlaces?: number;
}

export function SvgNumberTicker({
    value = 0,
    x,
    y,
    className = "",
    decimalPlaces = 0,
}: SvgNumberTickerProps) {
    const motionValue = useMotionValue(0);
    const springValue = useSpring(motionValue, {
        damping: 30,
        stiffness: 100,
    });

    useEffect(() => {
        motionValue.set(value || 0);
    }, [motionValue, value]);

    const displayValue = useTransform(
        springValue,
        (latest) =>
            `${Intl.NumberFormat("en-US", {
                minimumFractionDigits: decimalPlaces,
                maximumFractionDigits: decimalPlaces,
            }).format(latest)}%`
    );

    return (
        <motion.tspan x={x} y={y} className={className}>
            {displayValue}
        </motion.tspan>
    );
}
