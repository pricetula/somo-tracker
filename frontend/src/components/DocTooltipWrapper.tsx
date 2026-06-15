import { getTooltipContent } from '@/lib/docs';
import { DocTooltip } from './ui/DocTooltip';

interface WrapperProps {
  slug: string;
  anchorId?: string;
  fallbackText?: string;
}

export function FeatureHelp({ slug, anchorId, fallbackText }: WrapperProps) {
  const summary = getTooltipContent(slug) || fallbackText || "View documentation.";
  return <DocTooltip summary={summary} slug={slug} anchorId={anchorId} />;
}
