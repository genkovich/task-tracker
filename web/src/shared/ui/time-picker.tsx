import * as React from "react";

import { cn } from "@/shared/lib/utils";
import { Button } from "@/shared/ui/button";
import { TimePickerInput } from "@/shared/ui/time-picker-input";

interface TimePickerProps {
  value: string;
  onChange: (time: string) => void;
  minuteStep?: number;
  className?: string;
}

function fmt(hours: number, minutes: number): string {
  return `${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}`;
}

const PRESETS = [
  { label: "10:00", time: "10:00" },
  { label: "13:00", time: "13:00" },
  { label: "17:00", time: "17:00" },
] as const;

const DEFAULT_SCROLL_TIME = "10:00";

export function generateSlots(step: number): string[] {
  const slots: string[] = [];
  for (let h = 0; h < 24; h++) {
    for (let m = 0; m < 60; m += step) {
      slots.push(fmt(h, m));
    }
  }
  return slots;
}

export function TimePicker({
  value,
  onChange,
  minuteStep = 5,
  className,
}: TimePickerProps) {
  const [h, m] = value ? value.split(":").map(Number) : [0, 0];
  const hourRef = React.useRef<HTMLInputElement>(null);
  const minuteRef = React.useRef<HTMLInputElement>(null);
  const scrollRef = React.useRef<HTMLDivElement>(null);
  const hasScrolled = React.useRef(false);

  const slots = React.useMemo(
    () => generateSlots(minuteStep),
    [minuteStep]
  );

  // Auto-scroll to current value or default (10:00)
  React.useEffect(() => {
    if (hasScrolled.current) return;
    const container = scrollRef.current;
    const target = value || DEFAULT_SCROLL_TIME;
    requestAnimationFrame(() => {
      const el = container?.querySelector(
        `[data-time="${target}"]`
      ) as HTMLElement | null;
      if (container && el) {
        container.scrollTop = el.offsetTop;
        hasScrolled.current = true;
      }
    });
  }, [value, slots]);

  // Scroll selected slot into view when value changes
  React.useEffect(() => {
    if (!value) return;
    const container = scrollRef.current;
    const el = container?.querySelector(
      `[data-time="${value}"]`
    ) as HTMLElement | null;
    if (container && el) {
      const elTop = el.offsetTop;
      const elBottom = elTop + el.offsetHeight;
      const viewTop = container.scrollTop;
      const viewBottom = viewTop + container.clientHeight;
      if (elTop < viewTop) {
        container.scrollTop = elTop;
      } else if (elBottom > viewBottom) {
        container.scrollTop = elBottom - container.clientHeight;
      }
    }
  }, [value]);

  return (
    <div className={cn("flex flex-col gap-2", className)}>
      {/* Segment inputs */}
      <div className="flex items-center gap-1">
        <TimePickerInput
          ref={hourRef}
          picker="hours"
          value={h}
          onChange={(newH) => onChange(fmt(newH, m))}
          onRightFocus={() => minuteRef.current?.focus()}
        />
        <span className="text-muted-foreground text-sm font-medium select-none">
          :
        </span>
        <TimePickerInput
          ref={minuteRef}
          picker="minutes"
          value={m}
          step={minuteStep}
          onChange={(newM) => onChange(fmt(h, newM))}
          onLeftFocus={() => hourRef.current?.focus()}
        />
      </div>

      {/* Preset buttons */}
      <div className="flex gap-1">
        {PRESETS.map((preset) => (
          <Button
            key={preset.time}
            type="button"
            variant={value === preset.time ? "default" : "ghost"}
            size="sm"
            className="h-7 flex-1 px-1 text-xs"
            onClick={() => onChange(preset.time)}
          >
            {preset.label}
          </Button>
        ))}
      </div>

      {/* Time slot list */}
      <div
        ref={scrollRef}
        className="relative h-[200px] overflow-y-auto scrollbar-thin"
      >
        {slots.map((slot) => (
          <button
            key={slot}
            type="button"
            data-time={slot}
            className={cn(
              "w-full px-3 py-1.5 text-left text-sm tabular-nums transition-colors rounded-sm",
              slot === value
                ? "bg-primary text-primary-foreground"
                : "hover:bg-accent hover:text-accent-foreground"
            )}
            onClick={() => onChange(slot)}
          >
            {slot}
          </button>
        ))}
      </div>
    </div>
  );
}
