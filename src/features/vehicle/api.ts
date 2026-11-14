import { useQuery } from '@tanstack/react-query'
import { api, type Schemas } from '../../shared/api/client.ts'

export type Vehicle = Schemas['Vehicle']
export type VehicleModule = Schemas['VehicleModule']
export type TelemetryView = Schemas['TelemetryView']

/** Life support per module (FR-80). */
export function useVehicle() {
  return useQuery({
    queryKey: ['vehicle'],
    queryFn: () => api<Vehicle>('/vehicle'),
    refetchInterval: 60_000,
  })
}

/** Telemetry of every module for the last 7 days (FR-84). */
export function useTelemetry() {
  return useQuery({
    queryKey: ['telemetry'],
    queryFn: () => api<{ telemetry: TelemetryView[] }>('/vehicle/telemetry').then((r) => r.telemetry),
    refetchInterval: 5 * 60_000,
  })
}
