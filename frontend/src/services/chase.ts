/**
 * Chase API service for Monika frontend
 *
 * This service provides methods to interact with the chase API endpoints
 * defined in backend/src/api/chase.py
 */

import { api } from '../lib/api'
import type {
  Chase,
  ChaseRoundRequest,
  ChaseRoundResponse,
  ObstacleCheckRequest,
  ObstacleResponse,
  ChaseEndRequest,
  ChaseCreateRequest,
  ChaseParticipantCreateRequest,
  ChaseParticipant,
} from '../types/chase'

/**
 * Chase Service Class
 * Handles all chase-related API calls
 */
export class ChaseService {
  private baseURL = '/chase'

  /**
   * Get chase session by ID
   * @param chaseId - The chase session ID
   * @returns Chase session data
   */
  async getChase(chaseId: string): Promise<Chase> {
    const response = await api.get<Chase>(`${this.baseURL}/${chaseId}`)
    return response.data
  }

  /**
   * Execute round actions for a chase
   * @param chaseId - The chase session ID
   * @param request - Round action request with actions for all participants
   * @returns Round resolution response
   */
  async executeRoundAction(chaseId: string, request: ChaseRoundRequest): Promise<ChaseRoundResponse> {
    const response = await api.post<ChaseRoundResponse>(
      `${this.baseURL}/${chaseId}/round`,
      request
    )
    return response.data
  }

  /**
   * Perform an obstacle check for a participant
   * @param chaseId - The chase session ID
   * @param request - Obstacle check request with participant and obstacle info
   * @returns Obstacle check result response
   */
  async performObstacleCheck(chaseId: string, request: ObstacleCheckRequest): Promise<ObstacleResponse> {
    const response = await api.post<ObstacleResponse>(
      `${this.baseURL}/${chaseId}/obstacle-check`,
      request
    )
    return response.data
  }

  /**
   * Generate obstacles for a chase
   * @param chaseId - The chase session ID
   * @returns Array of generated obstacle responses
   */
  async generateObstacles(chaseId: string): Promise<ObstacleResponse[]> {
    const response = await api.post<ObstacleResponse[]>(
      `${this.baseURL}/${chaseId}/obstacles/generate`
    )
    return response.data
  }

  /**
   * End a chase session
   * @param chaseId - The chase session ID
   * @param request - Optional end request with reason and fail-forward scene
   */
  async endChase(chaseId: string, request?: ChaseEndRequest): Promise<void> {
    await api.post(`${this.baseURL}/${chaseId}/end`, request || {})
  }

  /**
   * Create a new chase session
   * @param request - Chase creation request
   * @returns Created chase session
   */
  async createChase(request: ChaseCreateRequest): Promise<Chase> {
    const response = await api.post<Chase>(`${this.baseURL}/start`, request)
    return response.data
  }

  /**
   * Add a participant to a chase
   * @param chaseId - The chase session ID
   * @param request - Participant creation request
   * @returns Created participant
   */
  async addParticipant(chaseId: string, request: ChaseParticipantCreateRequest): Promise<ChaseParticipant> {
    const response = await api.post<ChaseParticipant>(
      `${this.baseURL}/${chaseId}/participants`,
      request
    )
    return response.data
  }

  /**
   * Remove a participant from a chase
   * @param chaseId - The chase session ID
   * @param participantId - The participant ID to remove
   */
  async removeParticipant(chaseId: string, participantId: string): Promise<void> {
    await api.delete(`${this.baseURL}/${chaseId}/participants/${participantId}`)
  }

  /**
   * Get all obstacles for a chase
   * @param chaseId - The chase session ID
   * @returns Array of obstacles
   */
  async getObstacles(chaseId: string): Promise<ObstacleResponse[]> {
    const response = await api.get<ObstacleResponse[]>(`${this.baseURL}/${chaseId}/obstacles`)
    return response.data
  }

  /**
   * Get all participants in a chase
   * @param chaseId - The chase session ID
   * @returns Array of participants
   */
  async getParticipants(chaseId: string): Promise<ChaseParticipant[]> {
    const response = await api.get<ChaseParticipant[]>(`${this.baseURL}/${chaseId}/participants`)
    return response.data
  }
}

// Export singleton instance
export const chaseService = new ChaseService()
