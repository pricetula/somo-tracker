'use client'

import * as React from 'react'

type FormProps = React.ComponentProps<'form'> & {
  children?: React.ReactNode
}

const Form: React.FC<FormProps> = (props) => {
  const { children, ...rest } = props
  return (
    <form {...rest}>
      {children}
    </form>
  )
}
Form.displayName = 'Form'

type FormFieldProps = React.ComponentProps<'div'> & {
  children?: React.ReactNode
  name?: string
  label?: string
  description?: string
}

const FormField = React.forwardRef<
  HTMLDivElement,
  FormFieldProps
>((props, ref) => {
  const { children, className, label, description, name, ...rest } = props
  const fieldState = { error: undefined } as any

  return (
    <div className={['mb-4', className].filter(Boolean).join(' ')}>
      {label && (
        <label className="text-sm font-medium text-muted-foreground mb-1">
          {label}
        </label>
      )}
      <div className="relative">{typeof children === 'function' ? children({ field: fieldState }) : children}</div>
      {fieldState.error && (
        <p className="text-xs text-destructive">{fieldState.error.message}</p>
      )}
    </div>
  )
})
FormField.displayName = 'FormField'

const FormItem = React.forwardRef<
  HTMLDivElement,
  React.ComponentProps<'div'>
>((props, ref) => {
  const { className, ...rest } = props
  return <div className={['mb-4', className].filter(Boolean).join(' ')} {...rest} />
})
FormItem.displayName = 'FormItem'

const FormLabel = React.forwardRef<
  HTMLLabelElement,
  React.ComponentProps<'label'>
>((props, ref) => {
  const { className, ...rest } = props
  return (
    <label ref={ref} className={['text-sm font-medium text-muted-foreground mb-1', className].filter(Boolean).join(' ')} {...rest} />
  )
})
FormLabel.displayName = 'FormLabel'

const FormControl = React.forwardRef<
  HTMLInputElement,
  React.ComponentProps<'input'>
>((props, ref) => {
  const { className, ...rest } = props
  return (
    <input
      ref={ref}
      className={[
        "flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-white file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/30 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:border-destructive aria-invalid:ring-2 aria-invalid:ring-destructive/20 dark:aria-invalid:border-destructive/50 dark:aria-invalid:ring-destructive/40 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        className,
      ].filter(Boolean).join(' ')}
      {...rest}
    />
  )
})
FormControl.displayName = 'FormControl'

const FormMessage = React.forwardRef<
  HTMLParagraphElement,
  React.ComponentProps<'p'> & {
    children?: React.ReactNode
    name?: string
  }
>((props, ref) => {
  const { children, className, name, ...rest } = props
  const fieldState = { error: undefined } as any
  if (!fieldState.error && !children) return null
  return (
    <p
      ref={ref}
      className={['text-xs text-destructive', className].filter(Boolean).join(' ')}
      {...rest}
    >
      {fieldState.error?.message ?? children}
    </p>
  )
})
FormMessage.displayName = 'FormMessage'

export { Form, FormField, FormItem, FormLabel, FormControl, FormMessage }
