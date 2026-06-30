export function KeelwaveMark({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 120 120"
      className={className}
      fill="none"
      aria-hidden="true"
    >
      <path fill="currentColor" d="M74 28v66q-22 0-22-21 0-29 22-45" />
      <path
        stroke="currentColor"
        strokeLinecap="round"
        strokeWidth="5"
        d="M24 84q15-9 31 0"
        opacity=".9"
      />
      <path
        stroke="currentColor"
        strokeLinecap="round"
        strokeWidth="5"
        d="M12 96q22-12 45 0"
        opacity=".65"
      />
      <path
        stroke="currentColor"
        strokeLinecap="round"
        strokeWidth="5"
        d="M0 108q29-16 59 0"
        opacity=".4"
      />
    </svg>
  )
}
