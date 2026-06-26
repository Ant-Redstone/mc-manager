// Decorative animated gradient orbs behind the app.
export function Background() {
  return (
    <div aria-hidden className="pointer-events-none fixed inset-0 -z-10 overflow-hidden">
      <div className="absolute -left-24 -top-32 size-[36rem] animate-float rounded-full bg-brand/20 blur-3xl" />
      <div
        className="absolute -right-32 top-1/3 size-[32rem] animate-float rounded-full bg-status-info/15 blur-3xl"
        style={{ animationDelay: '-7s' }}
      />
      <div
        className="absolute -bottom-40 left-1/4 size-[34rem] animate-float rounded-full bg-status-success/10 blur-3xl"
        style={{ animationDelay: '-14s' }}
      />
    </div>
  )
}
