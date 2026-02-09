import { Button, Space, Typography } from '@arco-design/web-react'
import { IconPlus, IconMenuUnfold, IconMenuFold } from '@arco-design/web-react/icon'

const { Title } = Typography

interface HeaderProps {
  onNewSession: () => void
  onToggleSidebar: () => void
  isSidebarOpen?: boolean
  isConnected?: boolean
}

function Header({ 
  onNewSession, 
  onToggleSidebar, 
  isSidebarOpen = true, 
  isConnected = false 
}: HeaderProps) {
  return (
    <div style={{ 
      display: 'flex', 
      alignItems: 'center', 
      justifyContent: 'space-between', 
      height: '100%', 
      padding: '0 24px',
      color: 'white'
    }}>
      <Space size="large">
        <Button 
          type="text" 
          icon={isSidebarOpen ? <IconMenuFold /> : <IconMenuUnfold />} 
          onClick={onToggleSidebar}
          style={{ color: 'white', fontSize: '20px' }}
        />
        <Title heading={4} style={{ margin: 0, color: 'white' }}>
          <span style={{ marginRight: '12px' }}>🤖</span> CC-Mono
        </Title>
        
        {/* 连接状态指示器 */}
        <span style={{ 
          fontSize: '12px', 
          padding: '2px 8px', 
          borderRadius: '12px', 
          backgroundColor: isConnected ? 'rgba(16, 185, 129, 0.3)' : 'rgba(239, 68, 68, 0.3)',
          border: `1px solid ${isConnected ? '#10b981' : '#ef4444'}`,
          color: isConnected ? '#a7f3d0' : '#fca5a5'
        }}>
          {isConnected ? 'RPC Connected' : 'RPC Offline'}
        </span>
      </Space>

      <Space size="medium">
        <Button 
          type="primary" 
          status="success" 
          icon={<IconPlus />} 
          onClick={onNewSession}
        >
          New Session
        </Button>
      </Space>
    </div>
  )
}

export default Header
