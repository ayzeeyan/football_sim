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

// TrainProdigyResponse gains the ovr field the endpoints return.
patch('src/services/api/prodigies.ts', [
  [
    "export interface TrainProdigyResponse {\n  status: string;\n  message: string;\n  gains?: Record<string, unknown>;",
    "export interface TrainProdigyResponse {\n  status: string;\n  message: string;\n  ovr?: number;\n  gains?: Record<string, unknown>;",
  ],
]);

// PlayerSheetModal: training result state.
patch('src/components/clubs/PlayerSheet/PlayerSheetModal.tsx', [
  [
    "  const [activeTab, setActiveTab] = useState<PlayerSheetTab>('overview');",
    "  const [activeTab, setActiveTab] = useState<PlayerSheetTab>('overview');\n  const [trainingResult, setTrainingResult] = useState('');",
  ],
]);
