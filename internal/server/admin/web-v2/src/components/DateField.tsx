import moment from 'moment'

interface DateFieldProps {
  date: string | null
  format?: string
}

export default function DateField({ date, format = 'DD MMM YYYY HH:mm' }: DateFieldProps) {
  if (!date) {
    return <span className="data text-xs text-muted-foreground">—</span>
  }

  return (
    <time dateTime={date} className="data text-xs text-muted-foreground">
      {moment(date).format(format)}
    </time>
  )
}
