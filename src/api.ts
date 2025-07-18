import { getToken } from './context/AuthContext';
const BASE = 'http://localhost:8082';
export async function login(username: string, password: string) {
  const res = await fetch('http://localhost:8080/v1/auth/login', { // adjust if HTTP façade needed
    method: 'POST',
    headers: {'Content-Type':'application/json'},
    body: JSON.stringify({ username, password })
  });
  return res.json();
}
export async function fetchProjects() {
  const token = getToken();
  const res = await fetch(`${BASE}/v1/projects`, { headers: { 'Authorization': `Bearer ${token}` } });
  return res.json();
}
export async function searchEvents(projectId: string, filters: Record<string,string>) {
  const token = getToken();
  const res = await fetch(`${BASE}/v1/search`, {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${token}`, 'Content-Type':'application/json' },
    body: JSON.stringify({ project_id: projectId, filters })
  });
  return res.json();
}
export async function fetchEventDetail(projectId: string, eventName: string, cursor?: string) {
  const token = getToken();
  const body: any = { project_id: projectId, event_name: eventName };
  if(cursor) body.cursor = cursor;
  const res = await fetch(`${BASE}/v1/detail`, {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${token}`, 'Content-Type':'application/json' },
    body: JSON.stringify(body)
  });
  return res.json();
}