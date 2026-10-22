# Somotracker — Public Landing Page

Astro + Tailwind CSS v4 marketing site. No React, minimal JS.

## Placeholders to fill

- `[DEADLINE]` — application close date
- `[FORM_ENDPOINT]` — form endpoint URL (e.g. Formspree)
- `[FOUNDER NAME]` — founder name
- `[FOUNDER STORY]` — 2-line plain story
- `[WHATSAPP_NUMBER_INTERNATIONAL_FORMAT]` — e.g. `2547XXXXXXXX`
- `[PILOT_SIZE]` — number of pilot schools
- `[POST_PILOT_PRICING_LINE]` — pricing line, or omit if undecided
- `[NUMBER]`, `[EMAIL]` — WhatsApp and email in footer
- Illustration slots: hero-adjacent, timetable, attendance, substitutions, founder photo
- `public/og-wordmark.png` — Open Graph image (white/ink wordmark)

## Build notes

- Confirm each FeatureCards item is demo-able before launch. Remove any that are not.
- Roadmap: only keep a feature in "Building" if it will ship by Term 1 2027; otherwise move to "Next".
- Form endpoint must return proper JSON; progressive enhancement script handles demo mode when `[FORM_ENDPOINT]` is a placeholder.
- Privacy Policy must cover children's data under Kenya's Data Protection Act 2019.

## Performance

- Self-hosted Inter Variable via `@fontsource-variable/inter`
- Total page weight target: < 500 KB
- No render-blocking third-party scripts; add Plausible or similar only if needed.
