import { List, Avatar, Card, Typography, Space } from '@arco-design/web-react'
import { IconUser, IconRobot, IconSettings } from '@arco-design/web-react/icon'
import { getRpcClient, type ChatMessage } from '../utils/rpcClient'
import { useState, useEffect } from 'react'

const { Text } = Typography

function MessageList() {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const rpcClient = getRpcClient()

  useEffect(() => {
    fetchMessages()

    // 订阅实时更新
    const handleEvent = (event: any) => {
      if (event.type === 'prompt_added') {
        const newMsg = rpcClient.mapAgentMessageToChatMessage(event.data.message)
        setMessages(prev => {
          // 避免重复：检查 ID
          if (prev.some(m => m.id === newMsg.id)) return prev
          return [...prev, newMsg]
        })
      } else if (event.type === 'turn_end') {
        const finalMsg = rpcClient.mapAgentMessageToChatMessage(event.data.message)
        setMessages(prev => {
          // 更新或添加：检查 ID
          const index = prev.findIndex(m => m.id === finalMsg.id)
          if (index !== -1) {
            const next = [...prev]
            next[index] = finalMsg
            return next
          }
          // 同时也清理之前可能还在 streaming 的相同 ID 消息或临时流消息
          const filtered = prev.filter(m => !m.isStreaming || m.id === finalMsg.id)
          const finalIndex = filtered.findIndex(m => m.id === finalMsg.id)
          if (finalIndex !== -1) {
            const next = [...filtered]
            next[finalIndex] = finalMsg
            return next
          }
          return [...filtered, finalMsg]
        })
      } else if (event.type === 'message_update') {
        const streamMsg = rpcClient.mapAgentMessageToChatMessage(event.data.message)
        streamMsg.isStreaming = true
        setMessages(prev => {
          // 查找是否已有该 ID 的消息（可能已经在列表中，或者是上一次的 delta）
          const index = prev.findIndex(m => m.id === streamMsg.id)
          if (index !== -1) {
            const next = [...prev]
            next[index] = streamMsg
            return next
          }
          // 如果没找到 ID，查找是否有一个正在 streaming 的占位符消息
          const streamingIndex = prev.findIndex(m => m.isStreaming && m.type === 'assistant')
          if (streamingIndex !== -1) {
            const next = [...prev]
            next[streamingIndex] = streamMsg
            return next
          }
          return [...prev, streamMsg]
        })
      } else if (event.type === 'tool_call') {
        const toolMsg = rpcClient.mapToolCallToChatMessage(event)
        setMessages(prev => {
          if (prev.some(m => m.id === toolMsg.id)) return prev
          return [...prev, toolMsg]
        })
      } else if (event.type === 'tool_result') {
        const toolMsg = rpcClient.mapToolResultToChatMessage(event)
        setMessages(prev => {
          const index = prev.findIndex(m => m.id === toolMsg.id)
          if (index !== -1) {
            const next = [...prev]
            next[index] = toolMsg
            return next
          }
          return [...prev, toolMsg]
        })
      }
    }

    rpcClient.onMessage(handleEvent)

    return () => {
      rpcClient.offMessage(handleEvent)
    }
  }, [])

  const fetchMessages = async () => {
    try {
      const msgList = await rpcClient.getCurrentSessionMessages()
      setMessages(prev => {
        // 合并现有消息和获取到的消息，以获取到的为准，但保留正在 streaming 的
        const combined = [...msgList]
        prev.forEach(p => {
          if (p.isStreaming && !combined.some(c => c.id === p.id)) {
            combined.push(p)
          }
        })
        return combined.sort((a, b) => {
          // 简单的排序逻辑，这里可能需要更精确的时间戳比较
          return 0 
        })
      })
    } catch (error) {
      console.error('Error fetching messages:', error)
    }
  }

  const getMessageStyle = (type: string) => {
    switch (type) {
      case 'user':
        return { border: '1px solid #bfdbfe', background: '#f0f7ff' }
      case 'tool':
        return { border: '1px solid #e5e7eb', background: '#f9fafb', fontStyle: 'italic' }
      default:
        return { border: '1px solid #e5e7eb', background: '#fff' }
    }
  }

  const getAvatarColor = (type: string) => {
    switch (type) {
      case 'user': return '#3b82f6'
      case 'tool': return '#86909c'
      default: return '#10b981'
    }
  }

  const getAvatarIcon = (type: string) => {
    switch (type) {
      case 'user': return <IconUser />
      case 'tool': return <IconSettings />
      default: return <IconRobot />
    }
  }

  const getDisplayName = (type: string) => {
    switch (type) {
      case 'user': return 'You'
      case 'tool': return 'System Tool'
      default: return 'CC-Mono Assistant'
    }
  }

  const getDisplayNameColor = (type: string) => {
    switch (type) {
      case 'user': return '#1e40af'
      case 'tool': return '#4b5563'
      default: return '#059669'
    }
  }

  return (
    <List
      dataSource={messages}
      render={(item) => (
        <List.Item style={{ marginBottom: '16px' }}>
          <Card 
            style={{ 
              boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
              ...getMessageStyle(item.type)
            }}
            bodyStyle={{ padding: '16px' }}
          >
            <div style={{ 
              display: 'flex', 
              alignItems: 'flex-start', 
              gap: '12px', 
              marginBottom: '12px' 
            }}>
              <Avatar 
                style={{ backgroundColor: getAvatarColor(item.type) }}
                size={32}
              >
                {getAvatarIcon(item.type)}
              </Avatar>
              
              <div style={{ flex: 1 }}>
                <div style={{ 
                  display: 'flex', 
                  justifyContent: 'space-between', 
                  alignItems: 'center', 
                  marginBottom: '8px' 
                }}>
                  <span style={{ 
                    fontSize: '15px', 
                    fontWeight: 'bold',
                    color: getDisplayNameColor(item.type)
                  }}>
                    {getDisplayName(item.type)}
                    {item.isStreaming && item.type === 'tool' && ' (In Progress...)'}
                  </span>
                  <Text type="secondary" style={{ fontSize: '12px' }}>
                    {item.timestamp}
                  </Text>
                </div>
                
                <div style={{ 
                  color: item.type === 'tool' ? '#4b5563' : '#1f2937', 
                  lineHeight: '1.7',
                  whiteSpace: 'pre-wrap',
                  fontSize: item.type === 'tool' ? '13px' : '14px'
                }}>
                  {item.content}
                </div>
              </div>
            </div>
          </Card>
        </List.Item>
      )}
    />
  )
}

export default MessageList
