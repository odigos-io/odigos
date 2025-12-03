'use client';

import React from 'react';
import { useOdigletPods } from '@/hooks/collectors/useOdigletPods';

function formatNumber(n: number | undefined) {
  if (n === undefined) return '-';
  if (n >= 1000) return n.toFixed(0);
  if (n >= 100) return n.toFixed(1);
  return n.toFixed(2);
}

export default function CollectorPodsPage() {
  const { pods, loading, error } = useOdigletPods();

  if (loading && pods.length === 0) {
    return <div style={{ padding: 16 }}>Loading collector pods metrics…</div>;
  }
  if (error) {
    return (
      <div style={{ padding: 16, color: 'tomato' }}>
        Failed to load collector pods metrics: {error.message}
      </div>
    );
  }

  return (
    <div style={{ padding: 16 }}>
      <h2 style={{ margin: 0, marginBottom: 12 }}>Data Collection Pods Metrics</h2>
      <div
        style={{
          overflowX: 'auto',
          border: '1px solid #3333',
          borderRadius: 8,
        }}
      >
        <table
          style={{
            width: '100%',
            borderCollapse: 'separate',
            borderSpacing: 0,
          }}
        >
          <thead>
            <tr>
              <th style={{ textAlign: 'left', padding: 12, borderBottom: '1px solid #3333' }}>Pod</th>
              <th style={{ textAlign: 'left', padding: 12, borderBottom: '1px solid #3333' }}>Node</th>
              <th style={{ textAlign: 'left', padding: 12, borderBottom: '1px solid #3333' }}>Ready</th>
              <th style={{ textAlign: 'left', padding: 12, borderBottom: '1px solid #3333' }}>Status</th>
              <th style={{ textAlign: 'right', padding: 12, borderBottom: '1px solid #3333' }}>Accepted r/s</th>
              <th style={{ textAlign: 'right', padding: 12, borderBottom: '1px solid #3333' }}>Dropped r/s</th>
              <th style={{ textAlign: 'right', padding: 12, borderBottom: '1px solid #3333' }}>Exporter OK r/s</th>
              <th style={{ textAlign: 'right', padding: 12, borderBottom: '1px solid #3333' }}>Exporter Err r/s</th>
              <th style={{ textAlign: 'left', padding: 12, borderBottom: '1px solid #3333' }}>Window</th>
              <th style={{ textAlign: 'left', padding: 12, borderBottom: '1px solid #3333' }}>Last Scrape</th>
            </tr>
          </thead>
          <tbody>
            {pods.map((p) => {
              const m = p.collectorMetrics;
              return (
                <tr key={p.name}>
                  <td style={{ padding: 12, borderBottom: '1px solid #3333' }}>
                    <div style={{ fontWeight: 600 }}>{p.name}</div>
                    <div style={{ fontSize: 12, opacity: 0.75 }}>
                      {p.namespace} • restarts: {p.restartsCount}
                    </div>
                  </td>
                  <td style={{ padding: 12, borderBottom: '1px solid #3333' }}>{p.nodeName}</td>
                  <td style={{ padding: 12, borderBottom: '1px solid #3333' }}>{p.ready}</td>
                  <td style={{ padding: 12, borderBottom: '1px solid #3333' }}>{p.status || '-'}</td>
                  <td style={{ padding: 12, borderBottom: '1px solid #3333', textAlign: 'right' }}>
                    {m ? formatNumber(m.metricsAcceptedRps) : '-'}
                  </td>
                  <td style={{ padding: 12, borderBottom: '1px solid #3333', textAlign: 'right' }}>
                    {m ? formatNumber(m.metricsDroppedRps) : '-'}
                  </td>
                  <td style={{ padding: 12, borderBottom: '1px solid #3333', textAlign: 'right' }}>
                    {m ? formatNumber(m.exporterSuccessRps) : '-'}
                  </td>
                  <td style={{ padding: 12, borderBottom: '1px solid #3333', textAlign: 'right' }}>
                    {m ? formatNumber(m.exporterFailedRps) : '-'}
                  </td>
                  <td style={{ padding: 12, borderBottom: '1px solid #3333' }}>{m?.window || '-'}</td>
                  <td style={{ padding: 12, borderBottom: '1px solid #3333' }}>{m?.lastScrape || '-'}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}


