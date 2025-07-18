import React, { useEffect, useState } from 'react';
import { fetchProjects } from '../api';
import { useNavigate } from 'react-router-dom';
export default function Dashboard() {
  const [projects, setProjects] = useState<any[]>([]);
  const nav = useNavigate();
  useEffect(()=>{ fetchProjects().then(setProjects); },[]);
  return (
    <div className="p-4">
      <h1 className="text-xl mb-4">Projects</h1>
      <ul>
        {projects.map(p=> <li key={p.id}><button onClick={()=>nav(`/search/${p.id}`)}>{p.name}</button></li>)}
      </ul>
    </div>
  );
}