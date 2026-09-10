import { useCallback, useEffect, useRef, useState } from 'react';

/** Toast state with auto-dismiss + timer cleanup (fixes leaked setTimeout). */
export function useToast(timeoutMs = 3800) {
  const [toast, setToast] = useState<string | null>(null);
  const timer = useRef<number | null>(null);

  const showToast = useCallback(
    (msg: string) => {
      setToast(msg);
      if (timer.current) window.clearTimeout(timer.current);
      timer.current = window.setTimeout(() => setToast(null), timeoutMs);
    },
    [timeoutMs],
  );

  useEffect(
    () => () => {
      if (timer.current) window.clearTimeout(timer.current);
    },
    [],
  );

  return { toast, showToast };
}
