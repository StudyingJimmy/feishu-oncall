import { ArrowRightOutlined, InfoCircleOutlined, ThunderboltFilled } from '@ant-design/icons'
import { Avatar, Button, Form, Input, Modal } from 'antd'
import { useEffect } from 'react'
import type { Tenant } from '../domain/types'

interface Props {
  tenant: Tenant | null
  initialDescription: string
  submitting: boolean
  onCancel: () => void
  onConfirm: (description: string) => Promise<void>
}

export default function CreateOncallModal({ tenant, initialDescription, submitting, onCancel, onConfirm }: Props) {
  const [form] = Form.useForm<{ description: string }>()

  useEffect(() => {
    if (tenant) form.setFieldsValue({ description: initialDescription })
  }, [tenant, initialDescription, form])

  const submit = async () => {
    try {
      const values = await form.validateFields()
      await onConfirm(values.description.trim())
      form.resetFields()
    } catch {
      // 表单校验错误交给 Form 展示；服务错误由工作台提示。
    }
  }

  return (
    <Modal
      className="create-oncall-modal"
      open={Boolean(tenant)}
      onCancel={onCancel}
      title={null}
      footer={null}
      width={520}
      destroyOnHidden
      maskClosable={!submitting}
      closable={!submitting}
    >
      <div className="modal-overline"><ThunderboltFilled /> 发起 ONCALL</div>
      <h2>确认联系租户</h2>
      <p className="modal-lead">补充问题描述，确认后会创建协作工单并执行拉群流程。</p>
      {tenant && <div className="selected-tenant-card"><Avatar style={{ background: tenant.color }}>{tenant.avatar}</Avatar><div><strong>{tenant.name}</strong><span>{tenant.domain} · {tenant.summary}</span></div></div>}
      <Form form={form} layout="vertical" requiredMark={false} onFinish={submit}>
        <Form.Item name="description" label="问题描述" rules={[{ required: true, whitespace: true, message: '请填写问题描述' }]}>
          <Input.TextArea autoSize={{ minRows: 4, maxRows: 8 }} maxLength={1000} showCount placeholder="请描述问题现象、影响范围和已尝试的处理方式" />
        </Form.Item>
        <div className="modal-info"><InfoCircleOutlined /> 当前为原型演示：确认后仅模拟拉群并创建本地工单。</div>
        <div className="modal-actions"><Button onClick={onCancel} disabled={submitting}>取消</Button><Button type="primary" htmlType="submit" loading={submitting} icon={<ArrowRightOutlined />}>确认拉群</Button></div>
      </Form>
    </Modal>
  )
}
