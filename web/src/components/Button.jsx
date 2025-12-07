
const Button = ({ children, variant = 'primary', className = '', ...props }) => {
    const baseClass = 'btn'
    const variantClass = variant === 'primary' ? 'btn-primary' : 'btn-ghost'

    return (
        <button className={`${baseClass} ${variantClass} ${className}`} {...props}>
            {children}
        </button>
    )
}

export default Button
