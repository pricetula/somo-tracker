"use client";

import { forwardRef, useEffect, useImperativeHandle, useRef } from "react";

export interface TurnstileRef {
  /** Request a fresh Turnstile token for the given action. */
  verify: () => Promise<string>;
}

interface Turnstile {
  render: (el: Element, opts: TurnstileOpts) => number;
  execute: (id: number | string, opts?: TurnstileExecuteOpts) => Promise<string>;
  reset: (id: number) => void;
}

interface TurnstileOpts {
  sitekey: string;
  action?: string;
}

interface TurnstileExecuteOpts {
  action?: string;
}

declare global {
  interface Window {
    turnstile?: Turnstile;
  }
}

/**
 * Renders an (invisible) Cloudflare Turnstile widget and exposes `verify()`,
 * which returns a fresh token. The widget renders into a hidden container so
 * no UI is shown unless Cloudflare decides a challenge is required.
 *
 * Requires `NEXT_PUBLIC_TURNSTILE_SITE_KEY` to be set.
 */
export const Turnstile = forwardRef<TurnstileRef, { siteKey: string }>(
  function Turnstile({ siteKey }, ref) {
    const containerRef = useRef<HTMLDivElement>(null);
    const widgetIdRef = useRef<number | null>(null);

    useImperativeHandle(
      ref,
      () => ({
        verify: async () => {
          const id = widgetIdRef.current;
          if (id == null) {
            throw new Error("Turnstile widget is not initialized");
          }
          const token = await window.turnstile!.execute(id, {
            action: "login",
          });
          return token;
        },
      }),
      [],
    );

    useEffect(() => {
      if (typeof window === "undefined" || !siteKey) {
        return;
      }

      const init = () => {
        const ts = window.turnstile;
        if (!ts) {
          return;
        }
        const id = ts.render(containerRef.current!, {
          sitekey: siteKey,
          action: "login",
        });
        widgetIdRef.current = id;
      };

      if (window.turnstile) {
        init();
        return;
      }

      const script = document.createElement("script");
      script.src =
        "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit";
      script.async = true;
      script.onload = init;
      document.head.appendChild(script);
    }, [siteKey]);

    return <div ref={containerRef} style={{ display: "none" }} />;
  },
);

Turnstile.displayName = "Turnstile";
