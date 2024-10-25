import type React from 'react'

export default function RequestsList({ sID, rID }: { sID: string; rID?: string }): React.JSX.Element {
  return (
    <>
      My navbar for {sID} / {rID ?? 'N/A'}
    </>
  )
}
