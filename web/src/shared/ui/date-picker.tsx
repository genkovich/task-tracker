import * as React from "react";
import { format } from "date-fns";
import { Calendar as CalendarIcon, Check, ChevronsUpDown } from "lucide-react";

import { cn } from "@/shared/lib/utils";
import { Button } from "@/shared/ui/button";
import { Calendar } from "@/shared/ui/calendar";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/shared/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/shared/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/shared/ui/select";
import { TimePicker } from "@/shared/ui/time-picker";

interface DatePickerProps {
  value?: Date;
  onChange: (date: Date | undefined) => void;
  placeholder?: string;
  className?: string;
  showTime?: boolean;
  minuteStep?: number;
  showDuration?: boolean;
  durationMinutes?: number | null;
  onDurationChange?: (minutes: number) => void;
  timezone?: string;
  onTimezoneChange?: (tz: string) => void;
  onOpenChange?: (open: boolean) => void;
}

const ALL_TIMEZONES = Intl.supportedValuesOf("timeZone");

export function DatePicker({
  value,
  onChange,
  placeholder = "Pick a date",
  className,
  showTime,
  minuteStep,
  showDuration,
  durationMinutes,
  onDurationChange,
  timezone,
  onTimezoneChange,
  onOpenChange: onOpenChangeProp,
}: DatePickerProps) {
  const [open, setOpen] = React.useState(false);

  const handleOpenChange = (nextOpen: boolean) => {
    setOpen(nextOpen);
    onOpenChangeProp?.(nextOpen);
  };
  const [tzOpen, setTzOpen] = React.useState(false);

  const browserTz = Intl.DateTimeFormat().resolvedOptions().timeZone;
  const defaultTz = timezone ?? browserTz;
  const [selectedTimezone, setSelectedTimezone] = React.useState(defaultTz);

  // Sync with prop changes
  React.useEffect(() => {
    setSelectedTimezone(timezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone);
  }, [timezone]);

  const handleDateSelect = (date: Date | undefined) => {
    if (!date) {
      onChange(undefined);
      if (!showTime) setOpen(false);
      return;
    }
    // Preserve existing time when selecting a new date
    if (showTime && value) {
      date.setHours(value.getHours(), value.getMinutes());
    }
    onChange(date);
    if (!showTime) setOpen(false);
  };

  const handleTimeSelect = (slot: string) => {
    const [hours, minutes] = slot.split(":").map(Number);
    const updated = value ? new Date(value) : new Date();
    updated.setHours(hours, minutes);
    onChange(updated);
  };

  const timeValue = value
    ? `${String(value.getHours()).padStart(2, "0")}:${String(value.getMinutes()).padStart(2, "0")}`
    : "";

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          className={cn(
            "justify-start text-left font-normal",
            !value && "text-muted-foreground",
            className
          )}
        >
          <CalendarIcon className="size-4" />
          {value ? format(value, showTime ? "PPP 'at' HH:mm" : "PPP") : <span>{placeholder}</span>}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <div className="flex flex-col sm:flex-row">
          <Calendar
            mode="single"
            selected={value}
            onSelect={handleDateSelect}
          />
          {showTime && value && (
            <div className="flex w-full flex-col gap-3 border-t p-3 sm:w-[180px] sm:border-t-0 sm:border-l">
              <TimePicker
                value={timeValue}
                onChange={handleTimeSelect}
                minuteStep={minuteStep ?? 15}
              />
              {showDuration && (
                <Select
                  value={String(durationMinutes ?? 30)}
                  onValueChange={(v) => onDurationChange?.(Number(v))}
                >
                  <SelectTrigger className="h-9 w-full text-xs">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="15">15 min</SelectItem>
                    <SelectItem value="30">30 min</SelectItem>
                    <SelectItem value="45">45 min</SelectItem>
                    <SelectItem value="60">1 hour</SelectItem>
                    <SelectItem value="90">1.5 hours</SelectItem>
                    <SelectItem value="120">2 hours</SelectItem>
                  </SelectContent>
                </Select>
              )}
              <Popover open={tzOpen} onOpenChange={setTzOpen}>
                <PopoverTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    role="combobox"
                    className="mt-auto h-7 w-full justify-between px-2 text-xs text-muted-foreground"
                  >
                    <span className="truncate">{selectedTimezone.replace(/_/g, " ")}</span>
                    <ChevronsUpDown className="ml-1 size-3 shrink-0 opacity-50" />
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-64 p-0" align="start">
                  <Command>
                    <CommandInput placeholder="Search timezone..." />
                    <CommandList>
                      <CommandEmpty>No timezone found</CommandEmpty>
                      <CommandGroup>
                        {ALL_TIMEZONES.map((tz) => (
                          <CommandItem
                            key={tz}
                            value={tz}
                            onSelect={() => {
                              setSelectedTimezone(tz);
                              onTimezoneChange?.(tz);
                              setTzOpen(false);
                            }}
                          >
                            <Check
                              className={cn(
                                "size-3",
                                selectedTimezone === tz ? "opacity-100" : "opacity-0"
                              )}
                            />
                            <span className="text-xs">{tz.replace(/_/g, " ")}</span>
                          </CommandItem>
                        ))}
                      </CommandGroup>
                    </CommandList>
                  </Command>
                </PopoverContent>
              </Popover>
            </div>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}
