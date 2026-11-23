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
  remove: (id: number) => void;
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
 * which returns a fresh token on each call.
 *
 * Guards against duplicate script tags and duplicate render() calls so the
 * component is safe to mount/unmount/remount without Cloudflare errors.
 */
export const Turnstile = forwardRef<TurnstileRef, { siteKey: string }>(
  function Turnstile({ siteKey }, ref) {
    const containerRef = useRef<HTMLDivElement>(null);
    const widgetIdRef = useRef<number | null>(null);
    const scriptAppendedRef = useRef(false);

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

      const container = containerRef.current;
      if (!container) {
        return;
      }

      const init = () => {
        const ts = window.turnstile;
        if (!ts) {
          return;
        }

        // Only render once per mount. If already rendered, reuse the id.
        if (widgetIdRef.current == null) {
          const id = ts.render(container, {
            sitekey: siteKey,
            action: "login",
          });
          widgetIdRef.current = id;
        }
      };

      if (window.turnstile) {
        init();
        return;
      }

      // Only append the script once across all mounts/remounts.
      if (!scriptAppendedRef.current) {
        const existing = document.querySelector(
          'script[src*="challenges.cloudflare.com/turnstile/v0/api.js"]',
        );
        if (!existing) {
          const script = document.createElement("script");
          script.src =
            "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit";
          script.async = true;
          script.onload = () => {
            scriptAppendedRef.current = true;
            init();
          };
          document.head.appendChild(script);
        } else {
          scriptAppendedRef.current = true;
          const scriptLoadInterval = setInterval(() => {
            if (window.turnstile) {
              clearInterval(scriptLoadInterval);
              init();
            }
          }, 50);
          setTimeout(() => clearInterval(scriptLoadInterval), 5000);
        }
      } else {
        init();
      }

      // Cleanup: reset the widget on unmount so remounts work cleanly.
      return () => {
        if (widgetIdRef.current != null && window.turnstile) {
          try {
            window.turnstile.remove(widgetIdRef.current);
          } catch {
            // ignore
          }
          widgetIdRef.current = null;
        }
      };
    }, [siteKey]);

    return <div ref={containerRef} style={{ display: "none" }} />;
  },
);

Turnstile.displayName = "Turnstile";
