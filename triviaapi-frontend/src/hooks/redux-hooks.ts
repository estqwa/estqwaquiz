import { TypedUseSelectorHook, useDispatch, useSelector } from 'react-redux';
import type { RootState, AppDispatch } from '../store';

// Типизированный хук useDispatch
export const useAppDispatch = () => useDispatch<AppDispatch>();

// Типизированный хук useSelector
export const useAppSelector: TypedUseSelectorHook<RootState> = useSelector; 