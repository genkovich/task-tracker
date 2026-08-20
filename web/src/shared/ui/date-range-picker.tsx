import * as React from "react";
import { format } from "date-fns";
import { Calendar as CalendarIcon, X } from "lucide-react";
import type { DateRange } from "react-day-picker";

import { cn } from "@/shared/lib/utils";
import { Button } from "@/shared/ui/button";
import { Calendar } from "@/shared/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/shared/ui/popover";

interface DateRangePickerProps {
  from?: Date;
  to?: Date;
  onChange: (from?: Date, to?: Date) => void;
  className?: string;
}

export function DateRangePicker({ from, to, onChange, className }: DateRangePickerProps) {
  const [open, setOpen] = React.useState(false);

  const selected: DateRange | undefined = from || to ? { from, to } : undefined;

  const hasValue = from || to;

  return (
    <div className={cn("flex items-center gap-1", className)}>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            className={cn(
              "justify-start text-left font-normal",
              !hasValue && "text-muted-foreground",
            )}
          >
            <CalendarIcon className="size-4" />
            {hasValue ? (
              <span>
                {from ? format(from, "MMM d") : "Start"} – {to ? format(to, "MMM d") : "End"}
              </span>
            ) : (
              <span>Date range</span>
            )}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0" align="start">
          <Calendar
            mode="range"
            selected={selected}
            onSelect={(range) => {
              onChange(range?.from, range?.to);
              if (range?.from && range?.to) {
                setOpen(false);
              }
            }}
            numberOfMonths={2}
          />
        </PopoverContent>
      </Popover>
      {hasValue && (
        <Button
          variant="ghost"
          size="icon"
          className="size-8"
          onClick={() => onChange(undefined, undefined)}
          aria-label="Clear date range"
        >
          <X className="size-3.5" />
        </Button>
      )}
    </div>
  );
}
