import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

export default function Login() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const { login } = useAuth()
  const nav = useNavigate()

  const handleSubmit = async () => {
    try {
      await login(username, password)
      nav('/dashboard')
    } catch (err) {
      // show an error toast in a real app
      console.error('Login failed', err)
    }
  }

  return (
    <div className="p-4 max-w-sm mx-auto">
      <h1 className="text-xl mb-4">Login</h1>
      <input
        placeholder="Username"
        value={username}
        onChange={e => setUsername(e.target.value)}
        className="border p-2 w-full mb-2"
      />
      <input
        type="password"
        placeholder="Password"
        value={password}
        onChange={e => setPassword(e.target.value)}
        className="border p-2 w-full mb-4"
      />
      <button
        className="bg-blue-500 text-white px-4 py-2"
        onClick={handleSubmit}
      >
        Sign In
      </button>
    </div>
  )
}