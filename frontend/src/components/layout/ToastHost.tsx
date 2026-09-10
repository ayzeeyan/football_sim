import React from 'react';
import { Info } from 'lucide-react';

export const ToastHost: React.FC<{ message: string | null }> = ({ message }) => {
  if (!message) return null;
  return (
    <div className="fixed bottom-6 inset-x-0 flex justify-center z-50 pointer-events-none px-4">
      <div
        role="status"
        aria-live="polite"
        className="bg-dugout text-bone border border-bone/40 px-5 py-2.5 shadow-raised font-semibold text-[14px] flex items-center gap-2 max-w-full"
      >
        <Info className="text-brass shrink-0" size={15} aria-hidden="true" />
        <span className="truncate">{message}</span>
      </div>
    </div>
  );
};
