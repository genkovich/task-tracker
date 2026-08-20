import { Check } from "lucide-react";

import { cn } from "@/shared/lib/utils";

import type { StepConfig, StepStatus } from "./types";

function getStepStatus(index: number, currentStep: number): StepStatus {
  if (index < currentStep) return "completed";
  if (index === currentStep) return "active";
  return "upcoming";
}

interface StepperProps {
  steps: StepConfig[];
  currentStep: number;
  className?: string;
}

function Stepper({ steps, currentStep, className }: StepperProps) {
  return (
    <div
      data-slot="stepper"
      role="list"
      aria-label="Progress"
      className={cn("flex items-center w-full", className)}
    >
      {steps.map((step, index) => {
        const status = getStepStatus(index, currentStep);
        const isLast = index === steps.length - 1;

        return (
          <div
            key={step.id}
            role="listitem"
            className={cn("flex items-center", !isLast && "flex-1")}
          >
            <div className="flex flex-col items-center gap-1">
              <StepIndicator step={index + 1} status={status} title={step.title} />
              <span
                className={cn(
                  "text-xs font-medium text-center max-w-20 leading-tight",
                  status === "active" && "text-primary",
                  status === "completed" && "text-muted-foreground",
                  status === "upcoming" && "text-muted-foreground/60",
                )}
              >
                {step.title}
              </span>
              {step.description && (
                <span className="text-[10px] text-muted-foreground/60 text-center max-w-20 leading-tight">
                  {step.description}
                </span>
              )}
            </div>

            {!isLast && (
              <div
                data-testid="step-connector"
                data-completed={index < currentStep}
                aria-hidden="true"
                className={cn(
                  "h-px flex-1 mx-2 mb-auto mt-4", // mt-4 = half of size-8 indicator
                  index < currentStep ? "bg-primary" : "bg-border",
                )}
              />
            )}
          </div>
        );
      })}
    </div>
  );
}

interface StepIndicatorProps {
  step: number;
  status: StepStatus;
  title: string;
}

function StepIndicator({ step, status, title }: StepIndicatorProps) {
  const label = `Step ${step}: ${title} — ${status === "completed" ? "completed" : status === "active" ? "active" : "pending"}`;

  if (status === "completed") {
    return (
      <div
        data-testid="step-completed"
        aria-label={label}
        className="flex size-8 items-center justify-center rounded-full bg-primary text-primary-foreground"
      >
        <Check className="size-4" aria-hidden="true" />
      </div>
    );
  }

  if (status === "active") {
    return (
      <div
        data-testid="step-active"
        aria-label={label}
        className="flex size-8 items-center justify-center rounded-full border-2 border-primary bg-background text-primary font-semibold text-sm"
      >
        {step}
      </div>
    );
  }

  return (
    <div
      data-testid="step-upcoming"
      aria-label={label}
      className="flex size-8 items-center justify-center rounded-full border border-border bg-background text-muted-foreground text-sm"
    >
      {step}
    </div>
  );
}

export { Stepper };
