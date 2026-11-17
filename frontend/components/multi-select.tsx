"use client"

import * as React from "react"
import { Button } from "@/components/ui/button"
import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxItem,
  ComboboxList,
  ComboboxValue,
  useComboboxAnchor,
} from "@/components/ui/combobox"

export interface MultiSelectOption {
  value: string
  label: string
  hint?: string // shown at the right of the option, e.g. a task count
}

/**
 * Searchable multi-select: type to filter, tick as many options as needed,
 * remove a choice with its chip's ×, or clear them all. No selection means "all".
 */
export function MultiSelect({
  options,
  value,
  onChange,
  placeholder,
  emptyText = "Nothing found",
  ariaLabel,
}: {
  options: MultiSelectOption[]
  value: string[]
  onChange: (value: string[]) => void
  placeholder: string
  emptyText?: string
  ariaLabel: string
}) {
  const anchor = useComboboxAnchor()
  // The combobox compares items by reference, so selected values map back to the same option objects.
  const byValue = React.useMemo(
    () => new Map(options.map((o) => [o.value.toLowerCase(), o])),
    [options]
  )
  const selected = value.map(
    (v) => byValue.get(v.toLowerCase()) ?? { value: v, label: v }
  )
  return (
    <div className="flex items-start gap-2">
      <Combobox
        items={options}
        multiple
        value={selected}
        onValueChange={(next: MultiSelectOption[]) =>
          onChange(next.map((o) => o.value))
        }
        itemToStringValue={(o: MultiSelectOption) => o.label}
      >
        <ComboboxChips ref={anchor} className="min-w-64 flex-1">
          <ComboboxValue>
            {(items: MultiSelectOption[]) =>
              items.map((o) => (
                <ComboboxChip key={o.value} aria-label={o.label}>
                  {o.label}
                </ComboboxChip>
              ))
            }
          </ComboboxValue>
          <ComboboxChipsInput
            placeholder={value.length ? "Add…" : placeholder}
            aria-label={ariaLabel}
          />
        </ComboboxChips>
        <ComboboxContent anchor={anchor}>
          <ComboboxEmpty>{emptyText}</ComboboxEmpty>
          <ComboboxList>
            {(o: MultiSelectOption) => (
              <ComboboxItem key={o.value} value={o}>
                <span className="truncate">{o.label}</span>
                {o.hint && (
                  <span className="ml-auto text-xs text-muted-foreground tabular-nums">
                    {o.hint}
                  </span>
                )}
              </ComboboxItem>
            )}
          </ComboboxList>
        </ComboboxContent>
      </Combobox>
      {value.length > 0 && (
        <Button variant="ghost" size="sm" onClick={() => onChange([])}>
          Clear
        </Button>
      )}
    </div>
  )
}
