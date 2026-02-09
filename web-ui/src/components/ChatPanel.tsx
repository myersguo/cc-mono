import { useState } from 'react'
import { Input, Button, Card, Space, Typography } from '@arco-design/web-react'
import { 
  IconSend, 
  IconAttachment, 
  IconClose, 
  IconRefresh 
} from '@arco-design/web-react/icon'
import MessageList from './MessageList'
import { getRpcClient } from '../utils/rpcClient'

const { TextArea } = Input
const { Paragraph } = Typography

interface ChatPanelProps {
  onSendMessage: (message: string) => void
  sessionId: string
}

function ChatPanel({ onSendMessage, sessionId }: ChatPanelProps) {
  const rpcClient = getRpcClient()
  const [inputValue, setInputValue] = useState('')
  const [isLoading, setIsLoading] = useState(false)

  const handleSend = async () => {
    if (!inputValue.trim()) return
    
    setIsLoading(true)
    const message = inputValue
    setInputValue('') // 先清空，避免重复点击
    
    try {
      await onSendMessage(message)
    } catch (error) {
      console.error('Error sending message:', error)
      setInputValue(message) // 出错时回填
    } finally {
      setIsLoading(false)
    }
  }

  const handleKeyPress = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && e.ctrlKey) {
      handleSend()
    }
  }

  return (
    <div style={{ 
      height: '100%', 
      display: 'flex', 
      flexDirection: 'column' 
    }}>
      {/* Chat Header */}
      <div style={{ 
        padding: '16px 24px', 
        background: '#fff', 
        borderBottom: '1px solid #e5e7eb', 
        boxShadow: '0 1px 2px rgba(0,0,0,0.05)' 
      }}>
        <div style={{ 
          display: 'flex', 
          justifyContent: 'space-between', 
          alignItems: 'center' 
        }}>
          <h3 style={{ margin: '0', color: '#1d2129' }}>
            Session: {sessionId === 'default' ? 'Default' : sessionId}
          </h3>
          <Button 
            icon={<IconRefresh />} 
            type="outline" 
            size="small"
          >
            Regenerate
          </Button>
        </div>
        <Paragraph type="secondary" style={{ fontSize: '12px', marginTop: '8px' }}>
          AI-powered coding assistant. Use /help to see available commands.
        </Paragraph>
      </div>

      {/* Messages Area */}
      <div style={{ 
        flex: 1, 
        overflow: 'auto', 
        padding: '16px 24px' 
      }}>
        <MessageList />
      </div>

      {/* Input Area */}
      <Card 
        style={{ 
          margin: '16px 24px', 
          border: '1px solid #d1d5db', 
          boxShadow: '0 4px 6px -1px rgba(0,0,0,0.1)' 
        }}
        bodyStyle={{ padding: '16px' }}
      >
        <Space direction="vertical" style={{ width: '100%' }} size="medium">
          <TextArea
            placeholder="Type your message here. Press Ctrl+Enter to send..."
            value={inputValue}
            onChange={(value) => setInputValue(value)}
            onKeyDown={handleKeyPress}
            rows={4}
            autoSize={{ minRows: 4, maxRows: 10 }}
            style={{ resize: 'none' }}
          />
          
          <div style={{ 
            display: 'flex', 
            justifyContent: 'space-between', 
            alignItems: 'center' 
          }}>
            <Space size="medium">
              <Button 
                icon={<IconAttachment />} 
                type="outline"
                disabled={isLoading}
              >
                Attach File
              </Button>
              <Button 
                icon={<IconClose />} 
                type="outline" 
                status="danger"
                onClick={() => setInputValue('')}
                disabled={isLoading || !inputValue}
              >
                Clear
              </Button>
            </Space>

            <Button 
              type="primary" 
              status="success"
              icon={<IconSend />} 
              onClick={handleSend} 
              disabled={!inputValue.trim() || isLoading}
              long
            >
              Send
            </Button>
          </div>
        </Space>
      </Card>
    </div>
  )
}

export default ChatPanel
