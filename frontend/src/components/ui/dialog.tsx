import React from "react"

interface DialogProps {
  open?: boolean
  onOpenChange?: (open: boolean) => void
  children: React.ReactNode
}

interface DialogContentProps {
  children: React.ReactNode
  className?: string
}

interface DialogHeaderProps {
  children: React.ReactNode
}

interface DialogTitleProps {
  children: React.ReactNode
}

interface DialogTriggerProps {
  asChild?: boolean
  children: React.ReactNode
}

const Dialog: React.FC<DialogProps> = ({ open, onOpenChange, children }) => {
  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div 
        className="fixed inset-0 bg-black bg-opacity-50" 
        onClick={() => onOpenChange?.(false)}
      />
      <div className="relative z-50">
        {children}
      </div>
    </div>
  )
}

const DialogContent: React.FC<DialogContentProps> = ({ children, className }) => {
  return (
    <div className={`bg-white rounded-lg shadow-lg max-w-md w-full mx-4 ${className || ''}`}>
      {children}
    </div>
  )
}

const DialogHeader: React.FC<DialogHeaderProps> = ({ children }) => {
  return (
    <div className="px-6 py-4 border-b">
      {children}
    </div>
  )
}

const DialogTitle: React.FC<DialogTitleProps> = ({ children }) => {
  return (
    <h2 className="text-lg font-semibold text-gray-900">
      {children}
    </h2>
  )
}

const DialogTrigger: React.FC<DialogTriggerProps> = ({ asChild, children }) => {
  return <>{children}</>
}

export { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } 