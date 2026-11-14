/** The AURA mark: deep-space disc crossed by the Aura Spectrum at 30°, in a cream orbit ring. */
export function AuraMark({ size = 32 }: { size?: number }) {
  return (
    <svg viewBox="0 0 64 64" width={size} height={size} aria-hidden="true">
      <defs>
        <clipPath id="aura-mark-disc">
          <circle cx="32" cy="32" r="28" />
        </clipPath>
      </defs>
      <circle cx="32" cy="32" r="28" fill="#0b1120" />
      <g clipPath="url(#aura-mark-disc)" transform="rotate(-30 32 32)">
        <rect x="-8" y="22" width="80" height="4" className="fill-spectrum-1" />
        <rect x="-8" y="26" width="80" height="4" className="fill-spectrum-2" />
        <rect x="-8" y="30" width="80" height="4" className="fill-spectrum-3" />
        <rect x="-8" y="34" width="80" height="4" className="fill-spectrum-4" />
        <rect x="-8" y="38" width="80" height="4" className="fill-spectrum-5" />
      </g>
      <circle cx="32" cy="32" r="30" fill="none" stroke="#f2e8d5" strokeWidth="2" />
    </svg>
  )
}
