import React, { createContext, useContext } from 'react'
import { login as apiLogin } from '../api'

let tokenStore = ''
export function setToken(tok: string) { tokenStore = tok }
export function getToken() { return tokenStore }

type AuthContextType = {
  login: (username: string, password: string) => Promise<void>
}

const AuthContext = createContext<AuthContextType>({
  login: async () => { /* no-op */ },
})

export const AuthProvider: React.FC<React.PropsWithChildren<{}>> = ({ children }) => {
  const loginFn = async (username: string, password: string) => {
    const { token } = await apiLogin(username, password)
    setToken(token)
  }

  return (
    <AuthContext.Provider value={{ login: loginFn }}>
      {children}
    </AuthContext.Provider>
  )
}

export const useAuth = () => useContext(AuthContext)