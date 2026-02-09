import { List, Avatar, Tag } from '@arco-design/web-react'
import { IconMessage, IconStar } from '@arco-design/web-react/icon'
import { getRpcClient, type SessionInfo } from '../utils/rpcClient'
import { useState, useEffect } from 'react'

interface SidebarProps {
  activeSession: string
  onSessionSelect: (id: string) => void
}

function Sidebar({ activeSession, onSessionSelect }: SidebarProps) {
  const [sessions, setSessions] = useState<SessionInfo[]>([])

  useEffect(() => {
    fetchSessions()
  }, [])

  const fetchSessions = async () => {
    const rpcClient = getRpcClient()
    
    try {
      const sessionList = await rpcClient.getSessions()
      setSessions(sessionList)
    } catch (error) {
      console.error('Error fetching sessions:', error)
    }
  }

  return (
    <div style={{ padding: '16px' }}>
      <div style={{ 
        fontSize: '14px', 
        fontWeight: 600, 
        color: '#334155', 
        marginBottom: '12px', 
        paddingLeft: '8px' 
      }}>
        Recent Sessions
      </div>
      
      <List
        size="small"
        dataSource={sessions}
        render={(item) => (
          <List.Item
            style={{ 
              cursor: 'pointer', 
              borderRadius: '8px', 
              marginBottom: '4px' 
            }}
            className={activeSession === item.id ? 'active-session' : ''}
            onClick={() => onSessionSelect(item.id)}
          >
            <List.Item.Meta
              avatar={
                <Avatar 
                  style={{ 
                    backgroundColor: activeSession === item.id 
                      ? '#165dff' 
                      : '#86909c',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center'
                  }}
                >
                  <IconMessage />
                </Avatar>
              }
              title={
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <span style={{ fontSize: '13px', fontWeight: '500' }}>{item.title}</span>
                  {item.favorite && <IconStar style={{ fontSize: '14px', color: '#f59e0b' }} />}
                </div>
              }
              description={
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <span style={{ fontSize: '12px', color: '#86909c' }}>{item.date}</span>
                  {item.unread && (
                    <Tag size="small" color="red">{item.unread}</Tag>
                  )}
                </div>
              }
            />
          </List.Item>
        )}
      />
    </div>
  )
}

export default Sidebar
