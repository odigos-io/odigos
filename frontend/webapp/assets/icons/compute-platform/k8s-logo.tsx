import React from 'react';
import { SVG } from '@/assets';
import { useTheme } from 'styled-components';

export const K8sLogo: SVG = ({ size = 16, rotate = 0, onClick }) => {
  const theme = useTheme();

  return (
    <svg width={size} height={size} viewBox='0 0 28 28' xmlns='http://www.w3.org/2000/svg' fill='none' style={{ transform: `rotate(${rotate}deg)` }} onClick={onClick}>
      <path fill={theme.colors.info} d='M0 14C0 6.26801 6.26801 0 14 0C21.732 0 28 6.26801 28 14C28 21.732 21.732 28 14 28C6.26801 28 0 21.732 0 14Z' />
      <path
        fill={theme.text.info}
        d='M21.5645 15.7452C21.5495 15.7452 21.5495 15.7452 21.5645 15.7452H21.5495C21.5345 15.7452 21.5195 15.7452 21.5195 15.7297C21.4894 15.7297 21.4594 15.7143 21.4294 15.7143C21.3243 15.6988 21.2342 15.6834 21.1441 15.6834C21.099 15.6834 21.054 15.6834 20.9939 15.668H20.9789C20.6635 15.6371 20.4082 15.6062 20.168 15.529C20.0629 15.4826 20.0328 15.4208 20.0028 15.3591C20.0028 15.3436 19.9878 15.3436 19.9878 15.3282L19.7926 15.2664C19.8827 14.556 19.8526 13.8147 19.6875 13.0888C19.5223 12.3629 19.237 11.6834 18.8465 11.0656L18.9967 10.9266V10.8958C18.9967 10.8185 19.0117 10.7413 19.0718 10.6641C19.252 10.4942 19.4772 10.3552 19.7475 10.1853C19.7926 10.1544 19.8376 10.139 19.8827 10.1081C19.9728 10.0618 20.0478 10.0154 20.1379 9.95367C20.153 9.93822 20.183 9.92278 20.213 9.89189C20.228 9.87645 20.2431 9.87645 20.2431 9.861C20.4533 9.67568 20.4983 9.3668 20.3482 9.16602C20.2731 9.05792 20.1379 8.99614 20.0028 8.99614C19.8827 8.99614 19.7776 9.04247 19.6724 9.11969C19.6574 9.13514 19.6574 9.13514 19.6424 9.15058C19.6124 9.16602 19.5974 9.19691 19.5673 9.21236C19.4922 9.28958 19.4322 9.35135 19.3721 9.42857C19.3421 9.45946 19.312 9.50579 19.267 9.53668C19.0568 9.76834 18.8615 9.95367 18.6663 10.0927C18.6213 10.1236 18.5762 10.139 18.5312 10.139C18.5011 10.139 18.4711 10.139 18.4411 10.1236H18.411L18.2308 10.2471C18.0356 10.0309 17.8254 9.84556 17.6152 9.66023C16.7142 8.93436 15.618 8.48649 14.4767 8.37838L14.4617 8.17761C14.4467 8.16216 14.4467 8.16216 14.4317 8.14672C14.3866 8.10039 14.3265 8.05405 14.3115 7.94595C14.2965 7.69884 14.3265 7.42085 14.3566 7.11197V7.09653C14.3566 7.05019 14.3716 6.98842 14.3866 6.94208C14.4016 6.84942 14.4166 6.75676 14.4317 6.64865V6.55598V6.50965C14.4317 6.23166 14.2214 6 13.9661 6C13.846 6 13.7259 6.06178 13.6358 6.15444C13.5457 6.2471 13.5006 6.37066 13.5006 6.50965V6.54054V6.6332C13.5006 6.74131 13.5157 6.83398 13.5457 6.92664C13.5607 6.97297 13.5607 7.0193 13.5757 7.08108V7.09653C13.6057 7.40541 13.6508 7.6834 13.6208 7.9305C13.6057 8.03861 13.5457 8.08494 13.5006 8.13127C13.4856 8.14672 13.4856 8.14672 13.4706 8.16216L13.4556 8.36293C13.1853 8.39382 12.915 8.42471 12.6447 8.48649C11.4884 8.74903 10.4673 9.35135 9.67141 10.2162L9.52125 10.1081H9.49121C9.46118 10.1081 9.43115 10.1236 9.40112 10.1236C9.35607 10.1236 9.31102 10.1081 9.26597 10.0772C9.07075 9.93822 8.87554 9.73745 8.6653 9.50579C8.63527 9.4749 8.60524 9.42857 8.56019 9.39768C8.50012 9.32046 8.44006 9.25869 8.36497 9.18147C8.34996 9.16602 8.31992 9.15058 8.28989 9.11969C8.27487 9.10425 8.25986 9.10425 8.25986 9.0888C8.16976 9.01158 8.04963 8.96525 7.92949 8.96525C7.79434 8.96525 7.6592 9.02703 7.58411 9.13514C7.43395 9.33591 7.479 9.64479 7.68923 9.83012C7.70425 9.83012 7.70425 9.84556 7.71926 9.84556C7.74929 9.861 7.76431 9.89189 7.79434 9.90734C7.88444 9.96911 7.95953 10.0154 8.04963 10.0618C8.09468 10.0772 8.13972 10.1081 8.18477 10.139C8.45507 10.3089 8.68032 10.4479 8.86052 10.6178C8.9356 10.695 8.9356 10.7722 8.9356 10.8494V10.8803L9.08577 11.0193C9.05573 11.0656 9.0257 11.0965 9.01068 11.1429C8.25986 12.3629 7.97454 13.7992 8.16976 15.2201L7.97454 15.2819C7.97454 15.2973 7.95953 15.2973 7.95953 15.3127C7.92949 15.3745 7.88444 15.4363 7.79434 15.4826C7.5691 15.5598 7.2988 15.5907 6.98345 15.6216H6.96843C6.92338 15.6216 6.86332 15.6216 6.81827 15.6371C6.72817 15.6371 6.63807 15.6525 6.53295 15.668C6.50292 15.668 6.47289 15.6834 6.44285 15.6834C6.42784 15.6834 6.41282 15.6834 6.39781 15.6988C6.12751 15.7606 5.96233 16.0232 6.00737 16.2703C6.05242 16.4865 6.24764 16.6255 6.4879 16.6255C6.53295 16.6255 6.56299 16.6255 6.60804 16.61C6.62305 16.61 6.63807 16.61 6.63807 16.5946C6.6681 16.5946 6.69814 16.5792 6.72817 16.5792C6.83329 16.5483 6.90837 16.5174 6.99847 16.471C7.04352 16.4556 7.08857 16.4247 7.13362 16.4093H7.14863C7.43395 16.3012 7.68923 16.2085 7.92949 16.1776H7.95953C8.04963 16.1776 8.10969 16.2239 8.15474 16.2548C8.16976 16.2548 8.16976 16.2703 8.18477 16.2703L8.39501 16.2394C8.7554 17.3822 9.44616 18.4015 10.3622 19.1429C10.5724 19.3127 10.7826 19.4517 11.0079 19.5907L10.9178 19.7915C10.9178 19.807 10.9328 19.807 10.9328 19.8224C10.9628 19.8842 10.9929 19.9614 10.9628 20.0695C10.8727 20.3012 10.7376 20.5328 10.5724 20.7954V20.8108C10.5424 20.8571 10.5123 20.888 10.4823 20.9344C10.4222 21.0116 10.3772 21.0888 10.3171 21.1815C10.3021 21.1969 10.2871 21.2278 10.2721 21.2587C10.2721 21.2741 10.2571 21.2896 10.2571 21.2896C10.1369 21.5521 10.227 21.8456 10.4523 21.9537C10.5123 21.9846 10.5724 22 10.6325 22C10.8127 22 10.9929 21.8764 11.083 21.7066C11.083 21.6911 11.098 21.6757 11.098 21.6757C11.113 21.6448 11.128 21.6139 11.143 21.5985C11.1881 21.4903 11.2031 21.4131 11.2331 21.3205C11.2482 21.2741 11.2632 21.2278 11.2782 21.1815C11.3833 20.8726 11.4584 20.6255 11.5935 20.4093C11.6536 20.3166 11.7287 20.3012 11.7887 20.2703C11.8038 20.2703 11.8038 20.2703 11.8188 20.2548L11.9239 20.0541C12.5846 20.3166 13.3054 20.4556 14.0262 20.4556C14.4617 20.4556 14.9122 20.4093 15.3327 20.3012C15.603 20.2394 15.8582 20.1622 16.1135 20.0695L16.2036 20.2394C16.2186 20.2394 16.2186 20.2394 16.2336 20.2548C16.3087 20.2703 16.3688 20.3012 16.4289 20.3938C16.549 20.61 16.6391 20.8726 16.7442 21.166V21.1815C16.7592 21.2278 16.7742 21.2741 16.7893 21.3205C16.8193 21.4131 16.8343 21.5058 16.8794 21.5985C16.8944 21.6293 16.9094 21.6448 16.9244 21.6757C16.9244 21.6911 16.9394 21.7066 16.9394 21.7066C17.0295 21.8919 17.2097 22 17.3899 22C17.45 22 17.5101 21.9846 17.5701 21.9537C17.6752 21.8919 17.7653 21.7992 17.7954 21.6757C17.8254 21.5521 17.8254 21.4131 17.7653 21.2896C17.7653 21.2741 17.7503 21.2741 17.7503 21.2587C17.7353 21.2278 17.7203 21.1969 17.7053 21.1815C17.6602 21.0888 17.6002 21.0116 17.5401 20.9344C17.5101 20.888 17.48 20.8571 17.45 20.8108V20.7954C17.2848 20.5328 17.1346 20.3012 17.0596 20.0695C17.0295 19.9614 17.0596 19.8996 17.0746 19.8224C17.0746 19.807 17.0896 19.807 17.0896 19.7915L17.0145 19.6062C17.8104 19.1274 18.4861 18.4479 18.9967 17.6139C19.267 17.1815 19.4772 16.7027 19.6274 16.2239L19.8076 16.2548C19.8226 16.2548 19.8226 16.2394 19.8376 16.2394C19.8977 16.2085 19.9427 16.1622 20.0328 16.1622H20.0629C20.3031 16.1931 20.5584 16.2857 20.8437 16.3938H20.8587C20.9038 16.4093 20.9488 16.4402 20.9939 16.4556C21.084 16.5019 21.1591 16.5328 21.2642 16.5637C21.2942 16.5637 21.3243 16.5792 21.3543 16.5792C21.3693 16.5792 21.3843 16.5792 21.3993 16.5946C21.4444 16.61 21.4744 16.61 21.5195 16.61C21.7447 16.61 21.9399 16.4556 22 16.2548C22 16.0695 21.8348 15.8224 21.5645 15.7452ZM14.6119 14.9884L13.9511 15.3127L13.2904 14.9884L13.1252 14.2625L13.5757 13.6757H14.3115L14.762 14.2625L14.6119 14.9884ZM18.5312 13.3822C18.6513 13.9073 18.6813 14.4324 18.6363 14.9421L16.3388 14.2625C16.1285 14.2008 16.0084 13.9846 16.0534 13.7683C16.0685 13.7066 16.0985 13.6448 16.1435 13.5985L17.9605 11.9151C18.2158 12.3475 18.411 12.8417 18.5312 13.3822ZM17.2398 10.9884L15.2726 12.4247C15.1074 12.5328 14.8822 12.5019 14.747 12.332C14.702 12.2857 14.6869 12.2239 14.6719 12.1622L14.5368 9.64479C15.5729 9.76834 16.519 10.2471 17.2398 10.9884ZM12.885 9.72201C13.0501 9.69112 13.2003 9.66023 13.3655 9.62934L13.2303 12.1004C13.2153 12.3166 13.0501 12.5019 12.8249 12.5019C12.7648 12.5019 12.6897 12.4865 12.6447 12.4556L10.6475 10.9884C11.2632 10.3552 12.029 9.92278 12.885 9.72201ZM9.92669 11.9151L11.7137 13.5521C11.8788 13.6911 11.8939 13.9537 11.7587 14.1236C11.7137 14.1853 11.6536 14.2317 11.5785 14.2471L9.25095 14.9421C9.16085 13.8919 9.3861 12.8263 9.92669 11.9151ZM9.52125 16.1004L11.9089 15.6834C12.1041 15.668 12.2843 15.807 12.3293 16.0077C12.3444 16.1004 12.3444 16.1776 12.3143 16.2548L11.3983 18.5251C10.5574 17.9691 9.88164 17.1197 9.52125 16.1004ZM15.0023 19.1737C14.6569 19.251 14.3115 19.2973 13.9511 19.2973C13.4256 19.2973 12.915 19.2046 12.4345 19.0502L13.6208 16.8417C13.7409 16.7027 13.9361 16.6409 14.1013 16.7336C14.1764 16.7799 14.2364 16.8417 14.2815 16.9035L15.4378 19.0502C15.3026 19.0965 15.1525 19.1274 15.0023 19.1737ZM17.9305 17.027C17.5551 17.6448 17.0596 18.139 16.4889 18.5251L15.5429 16.1931C15.4978 16.0077 15.5729 15.807 15.7531 15.7143C15.8132 15.6834 15.8883 15.668 15.9633 15.668L18.366 16.0849C18.2759 16.4247 18.1257 16.7336 17.9305 17.027Z'
      />
    </svg>
  );
};
