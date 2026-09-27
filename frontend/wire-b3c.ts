const fs = require('fs');

function patch(path, anchors) {
  let s = fs.readFileSync(path, 'utf8');
  const eol = s.includes('\r\n') ? '\r\n' : '\n';
  for (const [from, to] of anchors) {
    const f = from.replace(/\n/g, eol);
    const t = to.replace(/\n/g, eol);
    if (!s.includes(f)) { console.error('anchor not found in ' + path + ': ' + JSON.stringify(from)); process.exit(1); }
    s = s.replace(f, t);
  }
  fs.writeFileSync(path, s);
  console.log('patched ' + path);
}

// NegotiationCard: viewer respond controls (Tier B3).
patch('src/components/transfers/TransfersTab/NegotiationCard.tsx', [
  [
    "import React from 'react';",
    "import React, { useState } from 'react';",
  ],
  [
    "export const NegotiationCard: React.FC<{ neg: TransferNegotiation; onPlayerClick: (id: string) => void }> = ({ neg, onPlayerClick }) => (",
    "export const NegotiationCard: React.FC<{\n  neg: TransferNegotiation;\n  onPlayerClick: (id: string) => void;\n  /** Tier B (B3): when provided, the viewer can improve or withdraw. */\n  onRespond?: (neg: TransferNegotiation, action: 'improve' | 'withdraw', amount?: number) => void;\n}> = ({ neg, onPlayerClick, onRespond }) => {",
  ],
  [
    "    <div\n      className={cx(\n        'p-4 rounded-xl border transition-colors',",
    "    <NegotiationCardBody neg={neg} onPlayerClick={onPlayerClick} onRespond={onRespond} />\n  );\n};\n\nconst NegotiationCardBody: React.FC<{\n  neg: TransferNegotiation;\n  onPlayerClick: (id: string) => void;\n  onRespond?: (neg: TransferNegotiation, action: 'improve' | 'withdraw', amount?: number) => void;\n}> = ({ neg, onPlayerClick, onRespond }) => {\n  const [amount, setAmount] = useState('');\n  return (\n    <div\n      className={cx(\n        'p-4 rounded-xl border transition-colors',",
  ],
]);
