import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { fetchEventDetail } from '../api';
export default function EventDetail() {
  const { projectId, eventName } = useParams();
  const [entry,set]=useState<any>(null);
  useEffect(()=>{ fetchEventDetail(projectId!,eventName!).then(data=>set(data.entry)); },[]);
  return entry ? (
    <div className="p-4">
      <h1 className="text-xl mb-4">{eventName}</h1>
      <pre>{JSON.stringify(entry,null,2)}</pre>
    </div>
  ) : <div>Loading…</div>;
}