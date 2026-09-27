import { ArrowRightOutlined, InfoCircleOutlined, ThunderboltFilled } from '@ant-design/icons'
import { Avatar, Button, Drawer, Form, Input } from 'antd'
import { useEffect } from 'react'
import type { Tenant } from '../domain/types'

interface Props {
  tenant: Tenant | null
  initialDescription: string
  submitting: boolean
  onCancel: () => void
  onConfirm: (description: string) => Promise<void>
}

export default function MobileCreateOncallDrawer({ tenant, initialDescription, submitting, onCancel, onConfirm }: Props) {
  const [form] = Form.useForm<{ description: string }>()

  useEffect(() => {
    if (tenant) form.setFieldsValue({ description: initialDescription })
  }, [tenant, initialDescription, form])

  const submit = async ({ description }: { description: string }) => {
    try {
      await onConfirm(description.trim())
      form.resetFields()
    } catch {
      // 服务错误由工作台提示；保留输入供用户重试。
    }
  }

  return (
    <Drawer
      className="mobile-oncall-drawer"
      placement="bottom"
      height="min(78dvh, 570px)"
      open={Boolean(tenant)}
      onClose={onCancel}
      closable={!submitting}
      maskClosable={!submitting}
      destroyOnHidden
      title={null}
      getContainer={false}
      rootStyle={{ position: 'absolute' }}
    >
      <div className="mobile-drawer-handle" />
      <div className="mobile-drawer-eyebrow"><ThunderboltFilled /> 发起 ONCALL</div>
      <h2>确认联系租户</h2>
      <p>描述具体问题，我们会按所选租户发起协作。</p>
      {tenant && <div className="mobile-drawer-tenant"><Avatar style={{ background: tenant.color }}>{tenant.avatar}</Avatar><span><strong>{tenant.name}</strong><small>{tenant.domain} · {tenant.summary}</small></span></div>}
      <Form form={form} layout="vertical" requiredMark={false} onFinish={submit}>
        <Form.Item name="description" label="问题描述" rules={[{ required: true, whitespace: true, message: '请填写问题描述' }]}>
          <Input.TextArea autoSize={{ minRows: 4, maxRows: 7 }} maxLength={1000} showCount placeholder="说明问题现象、影响范围和已尝试的操作" />
        </Form.Item>
        <div className="mobile-drawer-note"><InfoCircleOutlined /> 演示模式：确认后仅模拟拉群和建单。</div>
        <div className="mobile-drawer-actions"><Button onClick={onCancel} disabled={submitting}>取消</Button><Button type="primary" htmlType="submit" loading={submitting} icon={<ArrowRightOutlined />}>确认拉群</Button></div>
      </Form>
    </Drawer>
  )
}
