import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { searchEvents } from '../api';
export default function Search() {
  const { projectId } = useParams();
  const [events,set]=useState<any[]>([]);
  const nav=useNavigate();
  useEffect(()=>{ searchEvents(projectId!,{}).then(data=>set(data.events)); },[projectId]);
  return (
    <div className="p-4">
      <h1 className="text-xl mb-4">Events</h1>
      <table className="min-w-full">
        <thead><tr><th>Name</th><th>Count</th><th>Last Seen</th></tr></thead>
        <tbody>
          {events.map(e=><tr key={e.name} onClick={()=>nav(`/detail/${projectId}/${e.name}`)}>
            <td>{e.name}</td><td>{e.count}</td><td>{new Date(e.last_seen/1e6).toLocaleString()}</td>
          </tr>)}
        </tbody>
      </table>
    </div>
  );
}