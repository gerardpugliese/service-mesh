import React, { useEffect, useState } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import './App.css';

function App() {
  const [requests, setRequests] = useState([]);
  const [circuitStates, setCircuitStates] = useState({
    'https://localhost:3001': 'closed',
    'https://localhost:3002': 'closed',
    'https://localhost:3003': 'closed',
  });
  const [stats, setStats] = useState({
    total: 0,
    success: 0,
    errors: 0,
    avgLatency: 0,
  });
  const [chartData, setChartData] = useState([]);
  const [activeFlows, setActiveFlows] = useState([]);

  useEffect(() => {
    const ws = new WebSocket('ws://localhost:8080/ws');

    ws.onmessage = (event) => {
      const requestEvent = JSON.parse(event.data);

      // Add animated flow
      const flowId = Math.random();
      setActiveFlows((prev) => [...prev, { id: flowId, upstream: requestEvent.upstream, status: requestEvent.status }]);
      
      // Remove flow after animation completes (1 second)
      setTimeout(() => {
        setActiveFlows((prev) => prev.filter((f) => f.id !== flowId));
      }, 1000);

      // Add to recent requests (keep last 20)
      setRequests((prev) => [requestEvent, ...prev].slice(0, 20));

      // Update circuit breaker state
      setCircuitStates((prev) => ({
        ...prev,
        [requestEvent.upstream]: requestEvent.circuitOpen ? 'open' : 'closed',
      }));

      // Update stats
      setStats((prev) => {
        const newTotal = prev.total + 1;
        const newSuccess =
          requestEvent.status === 'success' ? prev.success + 1 : prev.success;
        const newErrors =
          requestEvent.status === 'error' ? prev.errors + 1 : prev.errors;
        const newAvgLatency =
          (prev.avgLatency * prev.total + requestEvent.latency) / newTotal;

        return {
          total: newTotal,
          success: newSuccess,
          errors: newErrors,
          avgLatency: Math.round(newAvgLatency),
        };
      });

      // Update chart data
      setChartData((prev) => {
        const now = new Date();
        const timestamp = now.toLocaleTimeString();

        if (prev.length === 0) {
          return [
            {
              timestamp,
              requests: 1,
              latency: requestEvent.latency,
            },
          ];
        }

        const lastEntry = prev[prev.length - 1];

        if (lastEntry.timestamp === timestamp) {
          return [
            ...prev.slice(0, -1),
            {
              ...lastEntry,
              requests: lastEntry.requests + 1,
              latency: Math.round(
                (lastEntry.latency + requestEvent.latency) / 2
              ),
            },
          ];
        } else {
          const newData = [
            ...prev,
            {
              timestamp,
              requests: 1,
              latency: requestEvent.latency,
            },
          ];
          return newData.slice(-30);
        }
      });
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    ws.onclose = () => {
      console.log('WebSocket closed');
    };

    return () => ws.close();
  }, []);

  return (
    <div className="App">
      <header className="header">
        <h1>Service Mesh Proxy Dashboard</h1>
      </header>

      <div className="container">
        {/* Stats */}
        <div className="stats">
          <div className="stat-card">
            <h3>Total Requests</h3>
            <p className="stat-value">{stats.total}</p>
          </div>
          <div className="stat-card">
            <h3>Success Rate</h3>
            <p className="stat-value">
              {stats.total > 0
                ? ((stats.success / stats.total) * 100).toFixed(1)
                : 0}
              %
            </p>
          </div>
          <div className="stat-card">
            <h3>Avg Latency</h3>
            <p className="stat-value">{stats.avgLatency}ms</p>
          </div>
          <div className="stat-card">
            <h3>Errors</h3>
            <p className="stat-value" style={{ color: 'red' }}>
              {stats.errors}
            </p>
          </div>
        </div>

        {/* Traffic Flow Visualization */}
        <div className="traffic-flow">
          <h2>Traffic Flow</h2>
          <div className="flow-diagram">
            {/* Client to Proxy */}
            <div className="flow-row">
              <div className="flow-box client">
                <span>Client</span>
              </div>
              <div className="flow-arrows-horizontal">
                {activeFlows.slice(0, 3).map((flow) => (
                  <div
                    key={flow.id}
                    className={`animated-arrow ${flow.status}`}
                  >
                    →
                  </div>
                ))}
              </div>
              <div className="flow-box proxy">
                <span>Proxy</span>
              </div>
            </div>

            {/* Proxy to Upstreams */}
            <div className="proxy-to-upstreams">
              <div className="downward-arrows">
                {activeFlows.map((flow) => (
                  <div
                    key={`down-${flow.id}`}
                    className={`animated-arrow-down ${flow.status}`}
                  >
                    ↓
                  </div>
                ))}
              </div>

              <div className="upstreams-row">
                {[1, 2, 3].map((num) => (
                  <div key={num} className="flow-box upstream">
                    <span>Upstream {num}</span>
                    <span className="port">:300{num}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* Charts */}
        <div className="charts">
          <div className="chart-container">
            <h2>Requests Over Time</h2>
            {chartData.length > 0 ? (
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={chartData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#444" />
                  <XAxis dataKey="timestamp" stroke="#90caf9" />
                  <YAxis stroke="#90caf9" />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#1a1f2e', border: '1px solid #1e88e5' }}
                    labelStyle={{ color: '#e0e0e0' }}
                  />
                  <Legend />
                  <Line
                    type="monotone"
                    dataKey="requests"
                    stroke="#4caf50"
                    dot={false}
                    strokeWidth={2}
                  />
                </LineChart>
              </ResponsiveContainer>
            ) : (
              <p style={{ color: '#90caf9', padding: '20px' }}>Waiting for data...</p>
            )}
          </div>

          <div className="chart-container">
            <h2>Latency Over Time</h2>
            {chartData.length > 0 ? (
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={chartData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#444" />
                  <XAxis dataKey="timestamp" stroke="#90caf9" />
                  <YAxis stroke="#90caf9" />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#1a1f2e', border: '1px solid #1e88e5' }}
                    labelStyle={{ color: '#e0e0e0' }}
                  />
                  <Legend />
                  <Line
                    type="monotone"
                    dataKey="latency"
                    stroke="#ffb74d"
                    dot={false}
                    strokeWidth={2}
                  />
                </LineChart>
              </ResponsiveContainer>
            ) : (
              <p style={{ color: '#90caf9', padding: '20px' }}>Waiting for data...</p>
            )}
          </div>
        </div>

        {/* Circuit Breaker States */}
        <div className="circuit-breakers">
          <h2>Circuit Breaker States</h2>
          <div className="breaker-cards">
            {Object.entries(circuitStates).map(([upstream, state]) => (
              <div key={upstream} className={`breaker-card ${state}`}>
                <p className="upstream-name">{upstream}</p>
                <p className="state">{state.toUpperCase()}</p>
                <p className="indicator">{state === 'closed' ? '✓' : '✗'}</p>
              </div>
            ))}
          </div>
        </div>

        {/* Live Requests */}
        <div className="live-requests">
          <h2>Live Requests</h2>
          <div className="request-list">
            {requests.length === 0 ? (
              <p>Waiting for requests...</p>
            ) : (
              requests.map((req, idx) => (
                <div key={idx} className={`request-item ${req.status}`}>
                  <span className="method">{req.method}</span>
                  <span className="path">{req.path}</span>
                  <span className="upstream">{req.upstream.split('//')[1]}</span>
                  <span className="latency">{req.latency}ms</span>
                  <span className="status">
                    {req.status === 'success' ? '✓' : '✗'}
                  </span>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export default App;