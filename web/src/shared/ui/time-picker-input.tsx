import * as React from "react";

import { Input } from "@/shared/ui/input";

interface TimePickerInputProps {
  value: number;
  onChange: (value: number) => void;
  picker: "hours" | "minutes";
  step?: number;
  onRightFocus?: () => void;
  onLeftFocus?: () => void;
}

const TimePickerInput = React.forwardRef<HTMLInputElement, TimePickerInputProps>(
  ({ value, onChange, picker, step = 1, onRightFocus, onLeftFocus }, ref) => {
    const inputRef = React.useRef<HTMLInputElement>(null);
    const [flag, setFlag] = React.useState("");
    const timerRef = React.useRef<ReturnType<typeof setTimeout> | null>(null);

    React.useImperativeHandle(ref, () => inputRef.current!, []);

    const max = picker === "hours" ? 23 : 59;

    const clamp = (v: number) => {
      if (v < 0) return max;
      if (v > max) return 0;
      return v;
    };

    const snap = (v: number) => {
      if (step <= 1) return v;
      return Math.round(v / step) * step % (max + 1);
    };

    const commitFlag = React.useCallback(
      (digit: string) => {
        const num = Number(digit);
        onChange(Math.min(num, max));
        setFlag("");
      },
      [onChange, max]
    );

    React.useEffect(() => {
      return () => {
        if (timerRef.current) clearTimeout(timerRef.current);
      };
    }, []);

    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Tab") return;

      if (e.key === "ArrowRight") {
        e.preventDefault();
        onRightFocus?.();
        return;
      }
      if (e.key === "ArrowLeft") {
        e.preventDefault();
        onLeftFocus?.();
        return;
      }

      if (e.key === "ArrowUp") {
        e.preventDefault();
        onChange(clamp(snap(value) + step));
        return;
      }
      if (e.key === "ArrowDown") {
        e.preventDefault();
        onChange(clamp(snap(value) - step));
        return;
      }

      if (e.key === "Backspace" || e.key === "Delete") {
        e.preventDefault();
        onChange(0);
        setFlag("");
        if (timerRef.current) clearTimeout(timerRef.current);
        return;
      }

      if (/^[0-9]$/.test(e.key)) {
        e.preventDefault();
        const digit = Number(e.key);

        if (flag === "") {
          // First digit
          if (picker === "hours") {
            if (digit >= 3) {
              // 3-9 can only be 03-09
              onChange(digit);
              setFlag("");
              onRightFocus?.();
              return;
            }
          } else {
            if (digit >= 6) {
              // 6-9 can only be 06-09 for minutes
              onChange(digit);
              setFlag("");
              onRightFocus?.();
              return;
            }
          }

          onChange(digit);
          setFlag(e.key);
          if (timerRef.current) clearTimeout(timerRef.current);
          timerRef.current = setTimeout(() => {
            commitFlag(`0${e.key}`);
          }, 2000);
        } else {
          // Second digit
          if (timerRef.current) clearTimeout(timerRef.current);
          const combined = Number(`${flag}${e.key}`);

          if (combined > max) {
            // Invalid combo — treat as single digit
            onChange(digit);
          } else {
            onChange(combined);
          }
          setFlag("");
          onRightFocus?.();
        }
      }
    };

    return (
      <Input
        ref={inputRef}
        type="text"
        inputMode="numeric"
        aria-label={picker === "hours" ? "Hours" : "Minutes"}
        role="spinbutton"
        aria-valuemin={0}
        aria-valuemax={max}
        aria-valuenow={value}
        className="w-12 text-center tabular-nums px-0 [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
        value={String(value).padStart(2, "0")}
        onKeyDown={handleKeyDown}
        onFocus={() => inputRef.current?.select()}
        onClick={() => inputRef.current?.select()}
        readOnly
      />
    );
  }
);

TimePickerInput.displayName = "TimePickerInput";

export { TimePickerInput };
export type { TimePickerInputProps };
