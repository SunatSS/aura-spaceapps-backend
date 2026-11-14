/** Stable display code for a body system by its catalog order: SYS-01 … SYS-07. */
export function systemCode(index: number): string {
  return `SYS-${String(index + 1).padStart(2, '0')}`
}
