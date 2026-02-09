import { useState, useEffect } from 'react'
import { Layout, Message } from '@arco-design/web-react'
import ChatPanel from './components/ChatPanel'
import Sidebar from './components/Sidebar'
import Header from './components/Header'
import { getRpcClient } from './utils/rpcClient'
import './App.css'

const { Header: ArcoHeader, Content, Sider } = Layout

function App() {
  const [activeSession, setActiveSession] = useState<string>('1')
  const [isSidebarOpen, setIsSidebarOpen] = useState(true)
  const [isConnected, setIsConnected] = useState(false)
  const rpcClient = getRpcClient()

  // 初始化 RPC 客户端连接
  useEffect(() => {
    setIsConnected(rpcClient.isOnline())
    
    rpcClient.onConnectionChange((connected) => {
      setIsConnected(connected)
      if (connected) {
        Message.success('Connected to CC-Mono RPC server')
      } else {
        Message.warning('Disconnected from CC-Mono server, reconnecting...')
      }
    })
    
    return () => {
      rpcClient.disconnect()
    }
  }, [])

  const handleSendMessage = async (message: string) => {
    try {
      await rpcClient.sendMessage(message)
    } catch (error) {
      console.error('Error sending message:', error)
    }
  }

  const handleNewSession = async () => {
    try {
      await rpcClient.newSession()
      const newId = 'session-' + Date.now()
      setActiveSession(newId)
      Message.success('New session created successfully')
    } catch (error) {
      console.error('Error creating new session:', error)
    }
  }

  return (
    <Layout className="app-layout">
      <ArcoHeader className="app-header">
        <Header 
          onNewSession={handleNewSession} 
          onToggleSidebar={() => setIsSidebarOpen(!isSidebarOpen)} 
          isConnected={isConnected}
        />
      </ArcoHeader>
      <Layout>
        {isSidebarOpen && (
          <Sider width={260} className="app-sidebar">
            <Sidebar 
              activeSession={activeSession} 
              onSessionSelect={setActiveSession} 
            />
          </Sider>
        )}
        <Content className="app-content">
          <ChatPanel 
            onSendMessage={handleSendMessage} 
            sessionId={activeSession}
          />
        </Content>
      </Layout>
    </Layout>
  )
}

export default App
