import { useParams } from "react-router-dom"
import GuidedSetup from "@/components/GuidedSetup"
import { useSetupScript } from "@/hooks/use-team-overview"

export default function Connect() {
  const { team } = useParams<{ team: string }>()
  const { script, state, retry } = useSetupScript(team)

  return (
    <GuidedSetup
      team={team}
      setupState={state}
      setupScript={script}
      onRetry={retry}
      showSteps={false}
    />
  )
}
