import React, { useMemo } from 'react';
import type { ProdigyData } from '../types';

/** Hexagonal PAC/SHO/PAS/DRI/DEF/PHY radar — pure presentational component. */
export const ProdigyRadar: React.FC<{ attributes: ProdigyData['attributes']; ovr: number }> = ({ attributes, ovr }) => {
  const size = 260;
  const center = size / 2;
  const radius = 95;

  const { gridPolygons, dataPolygon, vertices, stats } = useMemo(() => {
    const stats = [
      { label: 'PAC', val: attributes.pace },
      { label: 'SHO', val: attributes.shooting },
      { label: 'PAS', val: attributes.passing },
      { label: 'DRI', val: attributes.dribbling },
      { label: 'DEF', val: attributes.defending },
      { label: 'PHY', val: attributes.physicality },
    ];
    const angleStep = (Math.PI * 2) / 6;
    const gridLevels = [0.25, 0.5, 0.75, 1.0];
    const gridPolygons = gridLevels.map((lvl) =>
      stats
        .map((_, i) => {
          const angle = i * angleStep - Math.PI / 2;
          return `${center + Math.cos(angle) * (radius * lvl)},${center + Math.sin(angle) * (radius * lvl)}`;
        })
        .join(' '),
    );
    const vertices = stats.map((st, i) => {
      const norm = Math.max(0.2, Math.min(1.0, st.val / 99));
      const angle = i * angleStep - Math.PI / 2;
      return { x: center + Math.cos(angle) * (radius * norm), y: center + Math.sin(angle) * (radius * norm), ...st, angle };
    });
    const dataPolygon = vertices.map((v) => `${v.x},${v.y}`).join(' ');
    return { gridPolygons, dataPolygon, vertices, stats };
  }, [attributes, center, radius]);

  return (
    <div>
      <div className="flex items-center justify-between border-b border-line pb-3 mb-2">
        <span className="eyebrow">Attribute matrix</span>
        <span className="font-mono text-[12px] text-brass font-semibold">OVR {ovr}</span>
      </div>
      <svg width={size} height={size} className="overflow-visible mx-auto" role="img" aria-label={`Attribute radar, overall ${ovr}`}>
        {gridPolygons.map((poly, idx) => (
          <polygon key={idx} points={poly} fill={idx === 3 ? 'rgba(234, 228, 214, 0.04)' : 'transparent'} stroke="rgba(234, 228, 214, 0.16)" strokeWidth="1" />
        ))}
        {stats.map((_, i) => {
          const angle = (i * Math.PI * 2) / 6 - Math.PI / 2;
          return (
            <line
              key={i}
              x1={center}
              y1={center}
              x2={center + Math.cos(angle) * radius}
              y2={center + Math.sin(angle) * radius}
              stroke="rgba(234, 228, 214, 0.16)"
              strokeWidth="1"
            />
          );
        })}
        <polygon points={dataPolygon} fill="rgba(199, 162, 58, 0.18)" stroke="#C7A23A" strokeWidth="2" />
        {vertices.map((v, i) => (
          <circle key={i} cx={v.x} cy={v.y} r={3.5} fill="#C7A23A" stroke="#0B0E0C" strokeWidth="1.5" />
        ))}
        {vertices.map((st, i) => {
          const labelDist = radius + 22;
          const x = center + Math.cos(st.angle) * labelDist;
          const y = center + Math.sin(st.angle) * labelDist;
          return (
            <g key={i} className="select-none">
              <text x={x} y={y - 3} fill="#9AA79D" fontSize="10" fontWeight="600" textAnchor="middle" dominantBaseline="middle">
                {st.label}
              </text>
              <text x={x} y={y + 9} fill="#EAE4D6" fontSize="11" fontWeight="700" fontFamily="'IBM Plex Mono', monospace" textAnchor="middle" dominantBaseline="middle">
                {st.val}
              </text>
            </g>
          );
        })}
      </svg>
    </div>
  );
};
