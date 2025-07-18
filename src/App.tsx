import React from 'react';
import { Routes, Route } from 'react-router-dom';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Search from './pages/Search';
import EventDetail from './pages/EventDetail';
export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Login />} />
      <Route path="/dashboard" element={<Dashboard />} />
      <Route path="/search/:projectId" element={<Search />} />
      <Route path="/detail/:projectId/:eventName" element={<EventDetail />} />
    </Routes>
  );
}